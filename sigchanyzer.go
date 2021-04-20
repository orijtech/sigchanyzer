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
		switch arg := call.Args[0].(type) {
		case *ast.Ident:
			if decl, ok := findDecl(arg).(*ast.CallExpr); ok {
				chanDecl = decl
			}
		case *ast.CallExpr:
			// Only signal.Notify(make(chan os.Signal), os.Interrupt) is safe,
			// conservatively treate others as not safe, see golang/go#45043
			if isBuiltinMake(pass.TypesInfo, arg) {
				return
			}
			chanDecl = arg
		}

		if chanDecl == nil || len(chanDecl.Args) != 1 {
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

func isBuiltinMake(info *types.Info, call *ast.CallExpr) bool {
	typVal := info.Types[call.Fun]
	if !typVal.IsBuiltin() {
		return false
	}
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return info.ObjectOf(fun).Name() == "make"
	default:
		return false
	}
}
