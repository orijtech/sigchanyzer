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

package signalchan

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `check for unbuffer channel of os.Signal, which can be at risk of missing the signal.`

// Analyzer describes struct slop analysis function detector.
var Analyzer = &analysis.Analyzer{
	Name:     "signalchan",
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
		if len(call.Args) != 1 {
			return
		}
		ch, ok := call.Args[0].(*ast.ChanType)
		if !ok {
			return
		}
		if pass.TypesInfo.Types[ch].Type.String() != "chan os.Signal" {
			return
		}

		id, ok := call.Fun.(*ast.Ident)
		if !ok || id == nil {
			return
		}
		if id.Name != "make" || !pass.TypesInfo.Types[id].IsBuiltin() {
			return
		}
		call.Args = append(call.Args, &ast.BasicLit{
			Kind:  token.INT,
			Value: "1",
		})

		var buf bytes.Buffer
		if err := format.Node(&buf, token.NewFileSet(), call); err != nil {
			return
		}
		pass.Report(analysis.Diagnostic{
			Pos:     call.Pos(),
			End:     call.End(),
			Message: "unbuffer os.Signal channel",
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: "change to buffer channel.",
				TextEdits: []analysis.TextEdit{{
					Pos:     call.Pos(),
					End:     call.End(),
					NewText: buf.Bytes(),
				}},
			}},
		})
	})
	return nil, nil
}
