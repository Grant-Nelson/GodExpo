package app

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/printer"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"sort"
)

var (
	skipVendor            = false
	skipTestFiles         = false
	matchPkgPaths         = false
	matchBuildConstraints = false

	buildConstraints = map[string]bool{
		`linux`: true,
		`amd64`: true,
	}
)

type Stats struct {
	fileCount         int
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

	assignedFuncComplexity int
	structSumOfWMC         int
	sumFuncComplexity      int
}

func NewStats() *Stats {
	return &Stats{}
}

func (s *Stats) Logf(format string, args ...interface{}) {
	if s == nil {
		return
	}
	if len(format) == 0 {
		fmt.Fprintln(os.Stderr)
		return
	}
	fmt.Fprintf(os.Stderr, `[!] `+format+"\n", args...)
}

func (s *Stats) RecordFile(fSet *token.FileSet, f *ast.File, path string) {
	if s == nil {
		return
	}

	s.fileCount++

	fLoc := calcLoc(fSet, f.FileStart, f.FileEnd, path)
	// s.Logf(`File LOC: %d <= %s`, fLoc, path)
	s.filesTotalLOC += fLoc

	if c := readBuildConstraint(f); c != nil {
		s.Logf(`File build constraint %s <= %s`, c.String(), path)
	}
}

func (s *Stats) RecordFunc(fSet *token.FileSet, fn *ast.FuncDecl, path string) {
	if s == nil {
		return
	}

	s.sumFuncComplexity += complexity(fn)
	loc := calcLoc(fSet, fn.Pos(), fn.End(), path)

	s.totalFuncs++
	s.totalLOC += loc
	if fn.Recv == nil || fn.Recv.List[0].Names == nil {
		s.skippedFuncs++
		s.skippedLOC += loc
	}
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
		return
	}

	s.assignedFuncComplexity += m.Complexity

	if len(assigned) >= 2 {
		s.Logf(`Over-assigned %s.%s.%s`, m.PkgName, m.StructName, m.FuncName)
		s.Logf(`  Path:       %s`, m.Pos.String())
		s.Logf(`  Complexity: %6d`, m.Complexity)
		for i, st := range assigned {
			s.Logf(`  %d. %s:%s @ %s`, i, st.PkgName, st.StructName, st.Pos.String())
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
	s.Logf(``)
	s.Logf(`Number of files:      %6d`, s.fileCount)
	s.Logf(`Total file LOC:       %6d`, s.filesTotalLOC)
	s.Logf(`Total types:          %6d`, s.totalTypes)
	s.Logf(`Used struct types:    %6d`, s.totalTypes-s.skippedTypes)
	s.Logf(`Total skipped types:  %6d`, s.skippedTypes)
	s.Logf(`  Skipped interfaces: %6d`, s.totalInterfaces)
	s.Logf(`  Skipped other:      %6d`, s.skippedTypes-s.totalInterfaces)
	s.Logf(`Total funcs:          %6d (%6d LOC)`, s.totalFuncs, s.totalLOC)
	s.Logf(`Used funcs:           %6d (%6d LOC)`, s.totalFuncs-s.skippedFuncs, s.totalLOC-s.skippedLOC)
	s.Logf(`Total skipped funcs:  %6d (%6d LOC)`, s.skippedFuncs, s.skippedLOC)
	s.Logf(``)

	s.Logf(`Unassigned funcs:          %6d`, s.unassignedFunc)
	s.Logf(`LOC from unassigned funcs: %6d`, s.totalUnassignedLOC)
	s.Logf(`Max unassigned func: %s.%s.%s`, s.maxUnassigned.PkgName, s.maxUnassigned.StructName, s.maxUnassigned.FuncName)
	s.Logf(`  Path:       %s`, s.maxUnassigned.Pos.String())
	s.Logf(`  Complexity: %6d`, s.maxUnassigned.Complexity)
	s.Logf(`  LOC:        %6d`, s.maxUnassigned.LOC)
	s.Logf(``)

	s.Logf(`Over-assigned funcs:           %6d`, s.overAssignFunc)
	s.Logf(`Sum of over-assignments:       %6d`, s.totalOverAssign)
	s.Logf(`Sum of over-assigned func LOC: %6d`, s.sumOverAssignLOC)
	s.Logf(`Sum of over-represented LOC:   %6d`, s.totalOverAssignLOC)
	s.Logf(`Max over-assigned: %s.%s.%s`, s.maxOverAssign.PkgName, s.maxOverAssign.StructName, s.maxOverAssign.FuncName)
	s.Logf(`  Path:             %s`, s.maxOverAssign.Pos.String())
	s.Logf(`  Over-assignments: %6d`, s.maxOverAssignCount)
	s.Logf(`  Complexity:       %6d`, s.maxOverAssign.Complexity)
	s.Logf(`  LOC:              %6d`, s.maxOverAssign.LOC)
	s.Logf(``)

	s.Logf(`Sum of all func complexity:  %6d`, s.sumFuncComplexity)
	s.Logf(`Sum of used func complexity: %6d`, s.assignedFuncComplexity)
	s.Logf(`Sum of struct WMC:           %6d`, s.structSumOfWMC)
	s.Logf(``)

	s.Logf(`Skipped Types (no underlying struct):`)
	const skippedUnderlyingLimit = 100
	sort.Strings(s.skippedUnderlying)
	for i, t := range s.skippedUnderlying {
		if i >= skippedUnderlyingLimit {
			s.Logf(`  ... (+%d more)`, len(s.skippedUnderlying)-skippedUnderlyingLimit)
			break
		}
		s.Logf(`  %d. %q`, i+1, t)
	}
}

func getPackagePath(fset *token.FileSet, f *ast.File) string {
	return path.Dir(filepath.ToSlash(fset.Position(f.Package).Filename))
}

func calcLoc(fset *token.FileSet, start, end token.Pos, fname string) int {
	fLine := fset.Position(start).Line
	eLine := fset.Position(end).Line
	loc := eLine - fLine + 1
	if loc < 0 {
		panic(fmt.Errorf(`got a negative Loc (%d - %d + 1 = %d) in %s`, eLine, fLine, loc, fname))
	}
	return loc
}

// See https://pkg.go.dev/cmd/go#hdr-Build_constraints
// And https://pkg.go.dev/go/build/constraint
func readBuildConstraint(f *ast.File) constraint.Expr {
	for _, group := range f.Comments {
		for _, line := range group.List {
			if line != nil {
				if expr, err := constraint.Parse(line.Text); err == nil {
					return expr
				}
			}
		}
	}
	return nil
}

func isBuildConstraint(tag string) bool {
	return buildConstraints[tag]
}
