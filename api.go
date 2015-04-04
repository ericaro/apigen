package apigen

import "go/ast"

var (
	//ast definition of an *js.Object often use in apigen
	JSObject = &ast.StarExpr{
		X: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "js"},
			Sel: &ast.Ident{Name: "Object"},
		},
	}
)

//Api type holds a whole api.
type Api struct {
	Name    string   // package name
	Imports []string // imports list of imports
	Types   []*Type  // list of all types to be defined
	Funcs   []*Func  // all funcs (methods and funcs)
}

type Type struct {
	Name       string
	Properties []*Property
}

type Property struct {
	Name string   // property name
	Type ast.Expr //expression defining a type
	JS   string   // name in js
}

type Func struct {
	Description  string
	ReceiverType ast.Expr                //type expr or nil if it's not a method
	ReceiverName string                  // either the receiver local name, or a global name to be used to make the call
	Name         string                  // function anme
	JS           string                  // javascript side name
	Params       *ast.FieldList          // arguments
	ResultType   ast.Expr                // result type
	Convert      func(ast.Expr) ast.Expr // a function that turn the call expression ( *js.Object) into the return type.
}
