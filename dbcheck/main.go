package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"log"
	"os"
	"strings"

	"github.com/kisielk/gotool"
	"golang.org/x/tools/go/loader"
)

// TODO: Look for interfaces rather than for specific type names.
var lookForTypes = []string{
	"upper.io/db.v2/lib/sqlbuilder.Selector",
	"upper.io/db.v2/lib/sqlbuilder.Inserter",
	"upper.io/db.v2/lib/sqlbuilder.Updater",
	"upper.io/db.v2/lib/sqlbuilder.Deleter",
}

func matchType(name string) bool {
	for _, t := range lookForTypes {
		if t == name {
			return true
		}
	}
	return false
}

// tagsFlag. Taken from https://github.com/kisielk/errcheck
type tagsFlag []string

func (f *tagsFlag) String() string {
	return fmt.Sprintf("%q", strings.Join(*f, " "))
}

func (f *tagsFlag) Set(s string) error {
	if s == "" {
		return nil
	}
	tags := strings.Split(s, " ")
	if tags == nil {
		return nil
	}
	for _, tag := range tags {
		if tag != "" {
			*f = append(*f, tag)
		}
	}
	return nil
}

func main() {
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	tags := tagsFlag{}
	flags.Var(&tags, "tags", "'tag list'")

	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatal("flags.Parse: %v", err)
	}

	ctx := build.Default

	for i := range tags {
		ctx.BuildTags = append(ctx.BuildTags, tags[i])
	}

	loadcfg := loader.Config{
		Build: &ctx,
	}

	paths := gotool.ImportPaths(flags.Args())

	rest, err := loadcfg.FromArgs(paths, true)

	if err != nil {
		log.Fatalf("could not parse arguments: %s", err)
	}
	if len(rest) > 0 {
		log.Fatalf("unhandled extra arguments: %v", rest)
	}

	program, err := loadcfg.Load()
	if err != nil {
		log.Fatalf("Load: %s", err)
	}

	for _, pkgInfo := range program.InitialPackages() {
		for _, f := range pkgInfo.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				if expr, ok := n.(*ast.ExprStmt); ok {
					exprFn := expr.X.(*ast.CallExpr)
					if tv, ok := pkgInfo.Info.Types[exprFn]; ok {
						if matchType(tv.Type.String()) {
							pos := program.Fset.Position(exprFn.Pos())

							var buf bytes.Buffer
							format.Node(&buf, program.Fset, exprFn)

							fmt.Printf("%s:%d:%d %s\n", pos.Filename, pos.Line, pos.Column, buf)
						}
					}
				}
				return true
			})
		}
	}

}
