package app

import (
	"fmt"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func parsePath(fset *token.FileSet, stats *Stats, file string) ([]Struct, []Method) {
	var structs []Struct
	var methods []Method

	if isDir(file) {
		filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if skipVendor && strings.HasSuffix(filepath.ToSlash(path), `/vendor`) {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.HasSuffix(path, ".go") {
				if skipTestFiles && strings.HasSuffix(path, "_test.go") {
					return nil
				}

				f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
				if err != nil {
					log.Fatal(err)
				}

				if matchBuildConstraints {
					if expr := readBuildConstraint(f); expr != nil {
						if !expr.Eval(isBuildConstraint) {
							return nil
						}
					}
				}

				fmt.Fprintf(os.Stderr, "[*] Analyzing: %s\n", path)

				stats.RecordFile(fset, f, path)
				structs = append(structs, findStructsFromFile(fset, f, stats)...)
				methods = append(methods, findMethodsFromFile(fset, f, stats, path)...)
			}
			return err
		})

		fmt.Println()
	} else {
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			log.Fatal(err)
		}

		structs = findStructsFromFile(fset, f, stats)
		methods = findMethodsFromFile(fset, f, stats, file)
	}

	return structs, methods
}
