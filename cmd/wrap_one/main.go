//====================================================================================================//
// Copyright 2019 Trail of Bits
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//====================================================================================================//

package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"

	"golang.org/x/tools/go/ast/astutil"
)

//====================================================================================================//

const onedgePath = "github.com/trailofbits/on-edge"

var funcBodies = []ast.Node{}
var wrapped = false

//====================================================================================================//

func main() {
	if len(os.Args) != 2 {
		error("expect one argument: filename")
	}

	filename := os.Args[1]

	fset := token.NewFileSet()
	root, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		error("%v", err)
	}

	if astutil.UsesImport(root, onedgePath) {
		return
	}

	node := astutil.Apply(root, func(c *astutil.Cursor) bool {
		return func() bool {
			funcDecl, ok := c.Parent().(*ast.FuncDecl)
			if !ok {
				return true
			}
			if !(funcDecl.Body == c.Node() &&
				funcDecl.Type.Results != nil &&
				len(funcDecl.Type.Results.List) == 1 &&
				funcDecl.Type.Results.List[0].Names == nil) {
				return true
			}
			resultType, ok := funcDecl.Type.Results.List[0].Type.(*ast.Ident)
			if !ok {
				return true
			}
			if !(resultType.Name == "error") {
				return true
			}
			c.Replace(wrapFuncBody(funcDecl.Body))
			funcBodies = append(funcBodies, c.Node())
			wrapped = true
			return true
		}() && func() bool {
			if len(funcBodies) != 1 {
				return true
			}
			funcLit, ok := c.Parent().(*ast.FuncLit)
			if !ok {
				return true
			}
			if !(funcLit.Body == c.Node()) {
				return true
			}
			funcBodies = append(funcBodies, c.Node())
			return true
		}() && func() bool {
			if len(funcBodies) != 1 {
				return true
			}
			returnStmt, ok := c.Parent().(*ast.ReturnStmt)
			if !ok {
				return true
			}
			assert(returnStmt.Results != nil, "%+v.Results != nil", returnStmt)
			assert(len(returnStmt.Results) == 1, "len(%+v.Results) == 1", returnStmt)
			c.Replace(wrapReturnResult(returnStmt.Results[0]))
			return true
		}()
	}, func(c *astutil.Cursor) bool {
		if len(funcBodies) >= 1 && funcBodies[len(funcBodies)-1] == c.Node() {
			funcBodies = funcBodies[:len(funcBodies)-1]
		}
		return true
	})

	newRoot, ok := node.(*ast.File)
	assert(ok, "%+v.(*ast.File)", node)

	assert(len(funcBodies) == 0, "len(%+v) == 0", funcBodies)

	if !wrapped {
		return
	}

	astutil.AddImport(fset, newRoot, onedgePath)

	dst, err := os.OpenFile(filename, os.O_WRONLY, 0)
	if err != nil {
		error("could not open '%s' for writing: %v", filename, err)
	}
	if err = dst.Truncate(0); err != nil {
		error("could not write to '%s': %v", filename, err)
	}
	if err = format.Node(dst, fset, newRoot); err != nil {
		error("could not write to '%s': %v", filename, err)
	}
}

//====================================================================================================//

func wrapFuncBody(body *ast.BlockStmt) *ast.BlockStmt {
	selectorExpr := ast.SelectorExpr{
		X:   ast.NewIdent("onedge"),
		Sel: ast.NewIdent("WrapFuncRError"),
	}
	result := ast.Field{
		Type: ast.NewIdent("error"),
	}
	results := ast.FieldList{
		List: []*ast.Field{&result},
	}
	funcType := ast.FuncType{
		Params:  &ast.FieldList{},
		Results: &results,
	}
	funcLit := ast.FuncLit{
		Type: &funcType,
		Body: body,
	}
	callExpr := ast.CallExpr{
		Fun:  &selectorExpr,
		Args: []ast.Expr{&funcLit},
	}
	returnStmt := ast.ReturnStmt{
		Results: []ast.Expr{&callExpr},
	}
	blockStmt := ast.BlockStmt{
		List: []ast.Stmt{&returnStmt},
	}
	return &blockStmt
}

//====================================================================================================//

func wrapReturnResult(result ast.Expr) ast.Expr {
	selectorExpr := ast.SelectorExpr{
		X:   ast.NewIdent("onedge"),
		Sel: ast.NewIdent("WrapError"),
	}
	callExpr := ast.CallExpr{
		Fun:  &selectorExpr,
		Args: []ast.Expr{result},
	}
	return &callExpr
}

//====================================================================================================//

func error(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "%s: "+format+"\n", append([]interface{}{os.Args[0]}, a...)...)
	os.Exit(1)
}

//====================================================================================================//

func assert(x bool, a ...interface{}) {
	if !x {
		if len(a) <= 0 {
			panic("assertion failure")
		} else {
			format := a[0].(string)
			panic(fmt.Errorf("assertion failure: "+format, a[1:]...))
		}
	}
}

//====================================================================================================//
