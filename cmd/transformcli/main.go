// handoff transform is a planned tool that automatically transforms
// handoff tests via go:generate into tests that can be run by the go standard toolchain
// but includes support for plugins and test suite setup/teardown.
//
// Helpful links:
// http://goast.yuroyoro.net/ to view asts
// https://xdg.me/rewriting-go-with-ast-transformation/
// https://eli.thegreenplace.net/2021/rewriting-go-source-code-with-ast-tooling/
// https://pkg.go.dev/golang.org/x/tools/go/ast/astutil
package main

import (
	"flag"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"log"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func main() {
	flag.Parse()

	files := flag.Args()

	for _, f := range files {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
		if err != nil {
			log.Fatal(err)
		}

		astutil.Apply(file, nil, func(c *astutil.Cursor) bool {
			n := c.Node()
			switch x := n.(type) {
			case *ast.FuncDecl:
				if isHandoffTest(x) {
					param := x.Type.Params.List[0]

					if !strings.HasPrefix(x.Name.Name, "Test") {
						x.Name.Name = "Test" + x.Name.Name
					}

					sel := param.Type.(*ast.SelectorExpr)

					pkg := sel.X.(*ast.Ident)
					structName := sel.Sel

					pkg.Name = "testing"
					structName.Name = "T"
				}
			}

			return true
		})

		printer.Fprint(os.Stdout, fset, file)

	}
}

func isHandoffTest(f *ast.FuncDecl) bool {
	params := f.Type.Params.List
	if len(params) != 1 {
		return false
	}

	sel := params[0].Type.(*ast.SelectorExpr)

	funcName := sel.X.(*ast.Ident).Name + "." + sel.Sel.Name

	return funcName == "handoff.TB"
}
