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
			//if info.IsDir() {
			//	if strings.HasSuffix(filepath.ToSlash(path), `/vendor`) {
			//		return filepath.SkipDir
			//	}
			//	return err
			//}
			//if strings.HasSuffix(path, "_test.go") {
			//	return err
			//}
			if strings.HasSuffix(path, ".go") {
				f, err := parser.ParseFile(fset, path, nil, 0)
				if err != nil {
					log.Fatal(err)
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
