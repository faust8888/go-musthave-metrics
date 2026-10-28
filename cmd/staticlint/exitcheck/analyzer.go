// Package exitcheck implements an analyzer that forbids a direct os.Exit
// call in the main function of package main.
package exitcheck

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// Analyzer reports a diagnostic for each direct os.Exit call inside func main
// of package main. Calls in other functions of the same package are allowed.
var Analyzer = &analysis.Analyzer{
	Name: "exitcheck",
	Doc:  "forbids a direct os.Exit call in main() of package main; generated files are ignored",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	for _, file := range pass.Files {
		if ast.IsGenerated(file) {
			continue
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != "main" || fn.Recv != nil || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if isOsExit(pass, call) {
					pass.Reportf(call.Pos(), "direct call to os.Exit in main is forbidden")
				}
				return true
			})
		}
	}
	return nil, nil
}

func isOsExit(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	obj, ok := pass.TypesInfo.ObjectOf(sel.Sel).(*types.Func)
	if !ok {
		return false
	}
	return obj.FullName() == "os.Exit"
}
