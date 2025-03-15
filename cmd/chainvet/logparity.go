// This file contains the logparity checker.

package main

import (
	"go/ast"
	"go/types"
)

func init() {
	register("logparity",
		"check parity of key-value args in log invocations",
		checkLogCall,
		callExpr)
}

// checkLogCall triggers the print-specific checks if the call invokes a print function.
func checkLogCall(f *File, node ast.Node) {
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	fun, ok := f.pkg.uses[sel.Sel].(*types.Func)
	if !ok {
		return
	}

	if fun.FullName() != "chain/log.Printkv" && fun.FullName() != "chain/log.Fatalkv" {
		return
	}

	// Form of arguments is (ctx, key1, val1, key2, val2, ...)
	narg := len(call.Args) - 1 // Subtract one for the context.
	if narg%2 == 1 && call.Ellipsis == 0 {
		f.Badf(call.Pos(), "odd number of arguments in call to %s.%s", sel.X, sel.Sel)
	}
	// TODO(kr): perhaps check the type of the keys
	// and limit them to string or fmt.Stringer or similar?
}
