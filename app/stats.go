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
	"regexp"
	"sort"
	"strings"
)

var (
	basePath              = ``
	skipVendor            = false
	skipTestFiles         = false
	matchPkgPaths         = false
	matchBuildConstraints = false

	buildConstraints = map[string]bool{
		`release`: true,
		`linux`:   true, // README.md says "To install golang in ubuntu" so we assume linux
		`amd64`:   true,
	}

	vendorPathRegex = regexp.MustCompile(`(?:^|/)vendor(?:$|/)`)
	endOfLineRegex  = regexp.MustCompile(`[ \t]*\n`)
	whiteSpaceRegex = regexp.MustCompile(`\s+`)
)

type Stats struct {
	totalFiles      int
	vendorFiles     int
	testFiles       int
	buildConFiles   int
	eligibleFiles   int
	buildConFileMap map[string]bool

	totalFileLoc    int
	vendorFileLoc   int
	testFileLoc     int
	buildConFileLoc int
	eligibleFileLoc int

	skippedTypes      int
	totalInterfaces   int
	totalTypes        int
	vendorTypes       int
	testTypes         int
	buildConTypes     int
	eligibleTypes     int
	skippedUnderlying map[string]int

	totalFuncs    int
	skippedFuncs  int
	vendorFuncs   int
	testFuncs     int
	buildConFuncs int
	eligibleFuncs int

	skippedFuncLoc  int
	totalFuncLoc    int
	vendorFuncLoc   int
	testFuncLoc     int
	buildConFuncLoc int
	eligibleFuncLoc int

	unassignedFunc     int
	totalUnassignedLoc int
	maxUnassigned      *Method

	overAssignFunc     int
	sumOverAssignLoc   int
	totalOverAssignLoc int
	totalOverAssign    int
	maxOverAssignCount int
	maxOverAssign      *Method

	assignedFuncComplexity int
	structSumOfWmc         int
	sumFuncComplexity      int
}

func NewStats(path string) *Stats {
	basePath = filepath.ToSlash(path)
	if strings.HasSuffix(basePath, `.go`) {
		basePath = filepath.Dir(basePath)
	}
	if !strings.HasSuffix(basePath, `/`) {
		basePath += `/`
	}
	s := &Stats{}
	if len(os.Args) > 0 {
		s.Logf(`Command: godExpo %s`, strings.Join(os.Args[1:], ` `))
		s.Logf(``)
	}
	return s
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

	s.totalFiles++
	path = trimBasePath(filepath.ToSlash(path))

	fLoc := calcLoc(fSet, f.FileStart, f.FileEnd, path)
	// s.Logf(`  File LOC: %d`, fLoc)
	s.totalFileLoc += fLoc

	eligible := true
	if pathInVendor(path) {
		s.vendorFiles++
		s.vendorFileLoc += fLoc
		// s.Logf(`  File in vendor`)
		eligible = false
	}

	if strings.HasSuffix(path, `_test.go`) {
		s.testFiles++
		s.testFileLoc += fLoc
		// s.Logf(`  File is a test file`)
		eligible = false
	}

	if expr := readBuildConstraint(f); expr != nil {
		tag := `match`
		if !expr.Eval(isBuildConstraint) {
			s.buildConFiles++
			s.buildConFileLoc += fLoc
			tag = `not a match`
			if s.buildConFileMap == nil {
				s.buildConFileMap = map[string]bool{}
			}
			s.buildConFileMap[path] = true
			eligible = false
		}
		s.Logf(`  File build constraint [%s] %q`, tag, expr.String())
	}

	if eligible {
		s.eligibleFiles++
		s.eligibleFileLoc += fLoc
	}
}

func (s *Stats) RecordFunc(fSet *token.FileSet, fn *ast.FuncDecl, path string) {
	if s == nil {
		return
	}

	path = trimBasePath(filepath.ToSlash(path))
	s.sumFuncComplexity += complexity(fn)
	loc := calcLoc(fSet, fn.Pos(), fn.End(), path)

	s.totalFuncs++
	s.totalFuncLoc += loc
	if fn.Recv == nil || fn.Recv.List[0].Names == nil {
		s.skippedFuncs++
		s.skippedFuncLoc += loc
	}

	eligible := true
	if pathInVendor(path) {
		s.vendorFuncs++
		s.vendorFuncLoc += loc
		eligible = false
	}
	if strings.HasSuffix(path, `_test.go`) {
		s.testFuncs++
		s.testFuncLoc += loc
		eligible = false
	}
	if s.buildConFileMap[path] {
		s.buildConFuncs++
		s.buildConFuncLoc += loc
		eligible = false
	}
	if eligible {
		s.eligibleFuncs++
		s.eligibleFuncLoc += loc
	}
}

func (s *Stats) RecordType(fSet *token.FileSet, t *ast.TypeSpec) {
	if s == nil {
		return
	}

	pos := fSet.Position(t.Pos())
	path := trimBasePath(filepath.ToSlash(pos.Filename))

	s.totalTypes++
	_, ok := t.Type.(*ast.StructType)
	if !ok {
		s.skippedTypes++
		if _, ok := t.Type.(*ast.InterfaceType); ok {
			s.totalInterfaces++
		} else {
			under := &bytes.Buffer{}
			printer.Fprint(under, fSet, t.Type)
			ut := under.String()
			ut = endOfLineRegex.ReplaceAllString(ut, `;`)
			ut = whiteSpaceRegex.ReplaceAllString(ut, ` `)
			ut = strings.ReplaceAll(ut, `{;`, `{`)
			ut = strings.ReplaceAll(ut, `;}`, ` }`)
			decl := fmt.Sprintf(`%s %s @ %s:%d`, t.Name.Name, ut, path, pos.Line)

			if s.skippedUnderlying == nil {
				s.skippedUnderlying = map[string]int{}
			}
			s.skippedUnderlying[decl]++
		}
	}

	eligible := true
	if pathInVendor(path) {
		s.vendorTypes++
		eligible = false
	}
	if strings.HasSuffix(path, `_test.go`) {
		s.testTypes++
		eligible = false
	}
	if s.buildConFileMap[path] {
		s.buildConTypes++
		eligible = false
	}
	if eligible {
		s.eligibleTypes++
	}
}

func (s *Stats) RecordMethodAssignment(m Method, assigned []Struct) {
	if s == nil {
		return
	}

	if len(assigned) <= 0 {
		s.unassignedFunc++
		s.totalUnassignedLoc += m.Loc
		if s.maxUnassigned == nil || s.maxUnassigned.Loc < m.Loc {
			s.maxUnassigned = &m
		}
		return
	}

	s.assignedFuncComplexity += m.Complexity

	if len(assigned) >= 2 {
		mPath := trimBasePath(filepath.ToSlash(m.Pos.Filename))
		s.Logf(`Over-assigned %s.%s.%s`, m.PkgName, m.StructName, m.FuncName)
		s.Logf(`  Path:       %s:%d`, mPath, m.Pos.Line)
		s.Logf(`  Complexity: %6d`, m.Complexity)
		for i, st := range assigned {
			stPath := trimBasePath(filepath.ToSlash(st.Pos.Filename))
			s.Logf(`  %d. %s:%s @ %s:%d`, i, st.PkgName, st.StructName, stPath, st.Pos.Line)
		}
		s.Logf(``)

		s.overAssignFunc++
		s.sumOverAssignLoc += m.Loc
		s.totalOverAssignLoc += (m.Loc * len(assigned))
		s.totalOverAssign += len(assigned)
		if s.maxOverAssign == nil || s.maxOverAssignCount < len(assigned) {
			s.maxOverAssignCount = len(assigned)
			s.maxOverAssign = &m
		} else if s.maxOverAssignCount == len(assigned) {
			if s.maxOverAssign.Loc < m.Loc {
				s.maxOverAssign = &m
			}
		}
	}
}

func (s *Stats) RecordFinishedStruct(st Struct) {
	if s == nil {
		return
	}
	s.structSumOfWmc += st.WMC
}

func (s *Stats) Print() {
	if s == nil {
		return
	}
	s.Logf(``)
	s.Logf(`Total files:     %6d (%6d LOC)`, s.totalFiles, s.totalFileLoc)
	s.Logf(`Vendor files:    %6d (%6d LOC)`, s.vendorFiles, s.vendorFileLoc)
	s.Logf(`Test files:      %6d (%6d LOC)`, s.testFiles, s.testFileLoc)
	s.Logf(`Build con files: %6d (%6d LOC)`, s.buildConFiles, s.buildConFileLoc)
	s.Logf(`Eligible files:  %6d (%6d LOC)`, s.eligibleFiles, s.eligibleFileLoc)
	s.Logf(``)

	s.Logf(`Total types:          %6d`, s.totalTypes)
	s.Logf(`Used struct types:    %6d`, s.totalTypes-s.skippedTypes)
	s.Logf(`Total skipped types:  %6d`, s.skippedTypes)
	s.Logf(`  Skipped interfaces: %6d`, s.totalInterfaces)
	s.Logf(`  Skipped other:      %6d`, s.skippedTypes-s.totalInterfaces)
	s.Logf(`Vendor files:         %6d`, s.vendorTypes)
	s.Logf(`Test types:           %6d`, s.testTypes)
	s.Logf(`Build con types:      %6d`, s.buildConTypes)
	s.Logf(`Eligible types:       %6d`, s.eligibleTypes)
	s.Logf(``)

	s.Logf(`Total funcs:     %6d (%6d LOC)`, s.totalFuncs, s.totalFuncLoc)
	s.Logf(`Used funcs:      %6d (%6d LOC)`, s.totalFuncs-s.skippedFuncs, s.totalFuncLoc-s.skippedFuncLoc)
	s.Logf(`Skipped funcs:   %6d (%6d LOC)`, s.skippedFuncs, s.skippedFuncLoc)
	s.Logf(`Vendor funcs:    %6d (%6d LOC)`, s.vendorFuncs, s.vendorFuncLoc)
	s.Logf(`Test funcs:      %6d (%6d LOC)`, s.testFuncs, s.testFuncLoc)
	s.Logf(`Build con funcs: %6d (%6d LOC)`, s.buildConFuncs, s.buildConFuncLoc)
	s.Logf(`Eligible funcs:  %6d (%6d LOC)`, s.eligibleFuncs, s.eligibleFuncLoc)
	s.Logf(``)

	s.Logf(`Unassigned funcs:          %6d`, s.unassignedFunc)
	s.Logf(`LOC from unassigned funcs: %6d`, s.totalUnassignedLoc)
	if s.maxUnassigned != nil {
		path := trimBasePath(filepath.ToSlash(s.maxUnassigned.Pos.Filename))
		s.Logf(`Max unassigned func: %s.%s.%s`, s.maxUnassigned.PkgName, s.maxUnassigned.StructName, s.maxUnassigned.FuncName)
		s.Logf(`  Path:       %s:%d`, path, s.maxUnassigned.Pos.Line)
		s.Logf(`  Complexity: %6d`, s.maxUnassigned.Complexity)
		s.Logf(`  LOC:        %6d`, s.maxUnassigned.Loc)
	}
	s.Logf(``)

	s.Logf(`Over-assigned funcs:           %6d`, s.overAssignFunc)
	s.Logf(`Sum of over-assignments:       %6d`, s.totalOverAssign)
	s.Logf(`Sum of over-assigned func LOC: %6d`, s.sumOverAssignLoc)
	s.Logf(`Sum of over-represented LOC:   %6d`, s.totalOverAssignLoc)
	if s.maxOverAssign != nil {
		path := trimBasePath(filepath.ToSlash(s.maxOverAssign.Pos.Filename))
		s.Logf(`Max over-assigned: %s.%s.%s`, s.maxOverAssign.PkgName, s.maxOverAssign.StructName, s.maxOverAssign.FuncName)
		s.Logf(`  Path:             %s:%d`, path, s.maxOverAssign.Pos.Line)
		s.Logf(`  Over-assignments: %6d`, s.maxOverAssignCount)
		s.Logf(`  Complexity:       %6d`, s.maxOverAssign.Complexity)
		s.Logf(`  LOC:              %6d`, s.maxOverAssign.Loc)
	}
	s.Logf(``)

	s.Logf(`Sum of all func complexity:  %6d`, s.sumFuncComplexity)
	s.Logf(`Sum of used func complexity: %6d`, s.assignedFuncComplexity)
	s.Logf(`Sum of struct WMC:           %6d`, s.structSumOfWmc)
	s.Logf(``)

	s.Logf(`Skipped Types (no underlying struct):`)
	const skippedUnderlyingLimit = 100
	decls := make([]string, len(s.skippedUnderlying))
	i := 0
	for d := range s.skippedUnderlying {
		decls[i] = d
		i++
	}
	sort.Strings(decls)
	for i, t := range decls {
		if i >= skippedUnderlyingLimit {
			s.Logf(`  ... (+%d more)`, len(s.skippedUnderlying)-skippedUnderlyingLimit)
			break
		}
		s.Logf(`  %3d. %s`, i+1, t)
	}
}

func trimBasePath(pos string) string {
	if strings.HasPrefix(pos, basePath) {
		return pos[len(basePath):]
	}
	return pos
}

func pathInVendor(path string) bool {
	return vendorPathRegex.MatchString(path)
}

func getPackagePath(fSet *token.FileSet, f *ast.File) string {
	return trimBasePath(path.Dir(filepath.ToSlash(fSet.Position(f.Package).Filename)))
}

func calcLoc(fSet *token.FileSet, start, end token.Pos, path string) int {
	fLine := fSet.Position(start).Line
	eLine := fSet.Position(end).Line
	loc := eLine - fLine + 1
	if loc < 0 {
		panic(fmt.Errorf(`got a negative Loc (%d - %d + 1 = %d) in %s`, eLine, fLine, loc, path))
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
