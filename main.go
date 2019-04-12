package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var gitRef = flag.String("gitref", "head^", "git reference to compare against")

type checker struct {
	fset    *token.FileSet
	fsetold *token.FileSet
}

func main() {
	flag.Parse()
	stats, err := gitStats(*gitRef)
	if err != nil {
		log.Fatal(err)
	}
	dirs := make(map[string]struct{})
	for _, stat := range stats {
		path := stat.oldPath
		if filepath.Ext(path) != ".go" {
			continue
		}
		dir := filepath.Dir(path)
		if i := strings.Index(dir, "internal/"); i > -1 {
			dir = dir[:i]
		}
		if i := strings.Index(dir, "vendor/"); i > -1 {
			dir = dir[:i]
		}
		if len(dir) == 0 {
			// this was a root internal or vendor folder
			continue
		}
		if dir == "." {
			dir = "./"
		}
		dirs[dir] = struct{}{}
	}

	fset := token.NewFileSet()
	rev := "head"
	for dir := range dirs {
		blobs, err := gitLsTreeGoBlobs(rev, dir)
		if err != nil {
			// TODO: handle when paths are non-existent anymore
			if err == os.ErrNotExist {
				log.Printf("warning: path %q was removed\n", dir)
				continue
			}
			log.Fatal(err)
		}
		api := newPublicAPI()
		for _, file := range blobs {
			fullpath := filepath.Join(dir, file)
			r, err := gitBlob(rev, fullpath)
			if err != nil {
				log.Fatal(err)
			}
			f, err := parser.ParseFile(fset, fullpath, r, parser.AllErrors)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(fullpath)
			ast.Walk(api, f)
			break
		}
	}
}
