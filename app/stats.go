package app

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/printer"
	"go/token"
	"sort"
)

type Stats struct {
	skippedTypes      int
	totalInterfaces   int
	totalTypes        int
	skippedUnderlying []string

	skippedFuncs  int
	totalFuncs    int
	skippedLOC    int
	totalLOC      int
	filesTotalLOC int

	unassignedFunc     int
	totalUnassignedLOC int
	maxUnassigned      Method

	overAssignFunc     int
	sumOverAssignLOC   int
	totalOverAssignLOC int
	totalOverAssign    int
	maxOverAssignCount int
	maxOverAssign      Method

	funcSumOfWMC   int
	structSumOfWMC int
	everyWMC       int
}

func NewStats() *Stats {
	return &Stats{}
}

func (s *Stats) Logf(format string, args ...interface{}) {
	if s == nil {
		return
	}
	if len(format) == 0 {
		fmt.Println()
		return
	}
	fmt.Printf(`[!] `+format+"\n", args...)
}

func (s *Stats) RecordFile(fSet *token.FileSet, f *ast.File, path string) {
	if s == nil {
		return
	}

	fLoc := calcLoc(fSet, f.FileStart, f.FileEnd, path)
	// s.Logf(`File LOC: %d <= %s`, fLoc, path)
	s.filesTotalLOC += fLoc

	if f.Doc != nil {
		for _, line := range f.Doc.List {
			if line != nil {
				if exp, err := constraint.Parse(line.Text); err != nil {
					s.Logf(`File build constraint %s <= %s`, exp.String(), path)
				}
			}
		}
	}
}

func (s *Stats) RecordFunc(fSet *token.FileSet, fn *ast.FuncDecl, path string) {
	if s == nil {
		return
	}

	s.everyWMC += complexity(fn)
	loc := calcLoc(fSet, fn.Pos(), fn.End(), path)

	s.totalFuncs++
	s.totalLOC += loc
	if fn.Recv == nil || fn.Recv.List[0].Names == nil {
		s.skippedFuncs++
		s.skippedLOC += loc
	}
}

func (s *Stats) RecordFinishedFunc(m Method) {
	if s == nil {
		return
	}

	s.funcSumOfWMC += m.Complexity
}

func (s *Stats) RecordType(fSet *token.FileSet, t *ast.TypeSpec) {
	if s == nil {
		return
	}

	s.totalTypes++
	_, ok := t.Type.(*ast.StructType)
	if !ok {
		s.skippedTypes++
		if _, ok := t.Type.(*ast.InterfaceType); ok {
			s.totalInterfaces++
		} else {
			under := &bytes.Buffer{}
			printer.Fprint(under, fSet, t)
			s.skippedUnderlying = append(s.skippedUnderlying, under.String())
		}
	}
}

func (s *Stats) RecordMethodAssignment(m Method, assigned []Struct) {
	if s == nil {
		return
	}

	if len(assigned) <= 0 {
		s.unassignedFunc++
		s.totalUnassignedLOC += m.LOC
		if s.maxUnassigned.LOC < m.LOC {
			s.maxUnassigned = m
		}
	} else if len(assigned) >= 2 {
		s.Logf(`Over-assigned %s:%s:%s @ %s`, m.PkgName, m.StructName, m.FuncName, m.Pos.String())
		s.Logf("\tComplexity: %d", m.Complexity)
		for i, st := range assigned {
			s.Logf("\t%d. %s:%s @ %s", i, st.PkgName, st.StructName, st.Pos.String())
		}
		s.Logf(``)

		s.overAssignFunc++
		s.sumOverAssignLOC += m.LOC
		s.totalOverAssignLOC += (m.LOC * len(assigned))
		s.totalOverAssign += len(assigned)
		if s.maxOverAssignCount < len(assigned) {
			s.maxOverAssignCount = len(assigned)
			s.maxOverAssign = m
		} else if s.maxOverAssignCount == len(assigned) {
			if s.maxOverAssign.LOC < m.LOC {
				s.maxOverAssign = m
			}
		}
	}
}

func (s *Stats) RecordFinishedStruct(st Struct) {
	if s == nil {
		return
	}

	s.structSumOfWMC += st.WMC
}

func (s *Stats) Print() {
	if s == nil {
		return
	}

	s.Logf(`Types:  total: %d, taken: %d, skipped: %d`, s.totalTypes, s.totalTypes-s.skippedTypes, s.skippedTypes)
	s.Logf(`Inter:  total: %d, skipped other: %d`, s.totalInterfaces, s.skippedTypes-s.totalInterfaces)
	s.Logf(`Funcs:  total: %d, taken: %d, skipped: %d`, s.totalFuncs, s.totalFuncs-s.skippedFuncs, s.skippedFuncs)
	s.Logf(`Fn LOC: total: %d, taken: %d, skipped: %d`, s.totalLOC, s.totalLOC-s.skippedLOC, s.skippedLOC)
	s.Logf(``)

	s.Logf(`Unassigned Funcs: %d, Total LOC: %d, File LOC: %d`, s.unassignedFunc, s.totalUnassignedLOC, s.filesTotalLOC)
	s.Logf(`Max Unassigned Funcs: %s:%s:%s, CC: %d, LOC: %d`,
		s.maxUnassigned.PkgName, s.maxUnassigned.StructName, s.maxUnassigned.FuncName, s.maxUnassigned.Complexity, s.maxUnassigned.LOC)
	s.Logf(``)

	s.Logf(`Over-assigned: %d, Total Over: %d, Sum LOC: %d, Total LOC: %d`, s.overAssignFunc, s.totalOverAssign, s.sumOverAssignLOC, s.totalOverAssignLOC)
	s.Logf(`Max Over-assigned: %s:%s:%s, Count: %d, CC: %d, LOC: %d`,
		s.maxOverAssign.PkgName, s.maxOverAssign.StructName, s.maxOverAssign.FuncName, s.maxOverAssignCount, s.maxOverAssign.Complexity, s.maxOverAssign.LOC)
	s.Logf(``)

	s.Logf(`Func WMC: %d, Struct WMC: %d, All WMC: %d`, s.funcSumOfWMC, s.structSumOfWMC, s.everyWMC)
	s.Logf(``)

	s.Logf(`Skipped Types (no underlying struct):`)
	sort.Strings(s.skippedUnderlying)
	for i, t := range s.skippedUnderlying {
		s.Logf("\t%d. %q", i+1, t)
	}
}
