// Copyright 2020 Orijtech, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sigchanyzer

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `check for unbuffered channel of os.Signal, which can be at risk of missing the signal.`

// Analyzer describes struct slop analysis function detector.
var Analyzer = &analysis.Analyzer{
	Name:     "sigchanyzer",
	Doc:      Doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}
	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		if !isSignalNotify(pass.TypesInfo, call) {
			return
		}
		var chanDecl *ast.CallExpr

		// track whether call.Args[0] is a *ast.CallExpr. This is true when make(chan os.Signal) is passed directly to signal.Notify
		inlinedChanMake := false

		switch arg := call.Args[0].(type) {
		case *ast.Ident:
			if decl, ok := findDecl(arg).(*ast.CallExpr); ok {
				chanDecl = decl
			}
		case *ast.CallExpr:
			chanDecl = arg
			inlinedChanMake = true
		}

		if chanDecl == nil || len(chanDecl.Args) != 1 {
			return
		}

		if inlinedChanMake && isMakeCallExpr(pass.TypesInfo, chanDecl) {
			return
		}

		chanDecl.Args = append(chanDecl.Args, &ast.BasicLit{
			Kind:  token.INT,
			Value: "1",
		})
		var buf bytes.Buffer
		if err := format.Node(&buf, token.NewFileSet(), chanDecl); err != nil {
			return
		}
		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: "misuse of unbuffered os.Signal channel as argument to signal.Notify",
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: "Change to buffer channel",
				TextEdits: []analysis.TextEdit{{
					Pos:     chanDecl.Pos(),
					End:     chanDecl.End(),
					NewText: buf.Bytes(),
				}},
			}},
		})
	})
	return nil, nil
}

func isSignalNotify(info *types.Info, call *ast.CallExpr) bool {
	check := func(id *ast.Ident) bool {
		obj := info.ObjectOf(id)
		return obj.Name() == "Notify" && obj.Pkg().Path() == "os/signal"
	}
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		return check(fun.Sel)
	case *ast.Ident:
		if fun, ok := findDecl(fun).(*ast.SelectorExpr); ok {
			return check(fun.Sel)
		}
		return false
	default:
		return false
	}
}

func findDecl(arg *ast.Ident) ast.Node {
	if arg.Obj == nil {
		return nil
	}
	switch as := arg.Obj.Decl.(type) {
	case *ast.AssignStmt:
		if len(as.Lhs) != len(as.Rhs) {
			return nil
		}
		for i, lhs := range as.Lhs {
			lid, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}
			if lid.Obj == arg.Obj {
				return as.Rhs[i]
			}
		}
	case *ast.ValueSpec:
		if len(as.Names) != len(as.Values) {
			return nil
		}
		for i, name := range as.Names {
			if name.Obj == arg.Obj {
				return as.Values[i]
			}
		}
	}
	return nil
}

func isMakeCallExpr(info *types.Info, call *ast.CallExpr) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}

	name := ident.Name

	isFirstArgOsSignalType := func(info *types.Info, a1 ast.Expr) bool {
		cType, ok := a1.(*ast.ChanType)
		if !ok {
			return false
		}

		value := cType.Value

		selExpr, ok := value.(*ast.SelectorExpr)
		if !ok {
			return false
		}

		obj := info.ObjectOf(selExpr.Sel)
		return obj.Pkg().Path() == "os" && obj.Name() == "Signal"
	}

	isSecondArgIntOne := func(a2 ast.Expr) bool {
		bLit, ok := a2.(*ast.BasicLit)
		if !ok {
			return false
		}

		kind := bLit.Kind
		value := bLit.Value

		return kind == token.INT && value == "1"
	}

	args := call.Args

	switch len(args) {
	case 1:
		return name == "make" && isFirstArgOsSignalType(info, call.Args[0])
	case 2:
		return name == "make" && isFirstArgOsSignalType(info, call.Args[0]) && isSecondArgIntOne(call.Args[1])
	}

	return true
}
