package apigen

import (
	"fmt"
	"go/ast"
	"go/token"
)

func File(api *Api) (file *ast.File) {

	file = &ast.File{
		Name:  &ast.Ident{Name: api.Name}, //       *Ident          // package name
		Decls: make([]ast.Decl, 0, 100),   //      []Decl          // top-level declarations; or nil
	}

	//add the imports
	file.Decls = append(file.Decls, ImportDecl(api.Imports))

	for _, ty := range api.Types {
		file.Decls = append(file.Decls, TypeDecl(ty))
		file.Decls = append(file.Decls, Ctor(ty))
	}

	for _, f := range api.Funcs {
		file.Decls = append(file.Decls, FuncDecl(f))
	}
	return
}

// ImportDecl generate an import Declaration
func ImportDecl(imports []string) (impDecl *ast.GenDecl) {
	impDecl = &ast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: token.Pos(1),
		Specs:  make([]ast.Spec, len(imports)),
	}
	for i, imp := range imports {
		impDecl.Specs[i] = &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf("%q", imp),
			},
		}
	}
	return
}

//Field convert a Property into an ast.Field
func Field(p *Property) (f *ast.Field) {
	f = new(ast.Field)
	f.Names = []*ast.Ident{&ast.Ident{Name: p.Name}}
	f.Type = p.Type
	f.Tag = &ast.BasicLit{
		Kind:  token.STRING,
		Value: fmt.Sprintf("`js:\"%s\"`", p.JS),
	}
	return
}

func TypeDecl(ty *Type) (g *ast.GenDecl) {
	g = new(ast.GenDecl)
	g.Tok = token.TYPE

	tspec := new(ast.TypeSpec)
	g.Specs = []ast.Spec{tspec}
	tspec.Name = &ast.Ident{Name: ty.Name}

	st := new(ast.StructType)
	tspec.Type = st

	st.Fields = new(ast.FieldList)
	st.Fields.List = make([]*ast.Field, 0, 1+len(ty.Properties))

	//always prepend the anonymous *js.Object
	st.Fields.List = append(st.Fields.List, &ast.Field{Type: JSObject})

	for _, p := range ty.Properties {
		st.Fields.List = append(st.Fields.List, Field(p))
	}
	return
}

func Ctor(j *Type) *ast.FuncDecl {
	v := "j" // the name of the argument name
	return &ast.FuncDecl{
		Name: &ast.Ident{Name: "new" + j.Name},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Names: []*ast.Ident{
							&ast.Ident{Name: v},
						},
						Type: JSObject,
					},
				},
			},
			Results: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Type: &ast.Ident{Name: j.Name},
					},
				},
			},
		},
		Body: &ast.BlockStmt{
			List: []ast.Stmt{
				&ast.ReturnStmt{
					Results: []ast.Expr{
						&ast.CompositeLit{
							Type: &ast.Ident{Name: j.Name},
							Elts: []ast.Expr{
								&ast.KeyValueExpr{
									Key:   &ast.Ident{Name: "Object"},
									Value: &ast.Ident{Name: v},
								},
							},
						},
					},
				},
			},
		},
	}

}

func FuncDecl(f *Func) (fd *ast.FuncDecl) {
	// receiver, params, returns, and body

	var receiver *ast.FieldList
	if f.ReceiverType != nil {
		receiver = &ast.FieldList{List: []*ast.Field{
			&ast.Field{
				Names: []*ast.Ident{&ast.Ident{Name: f.ReceiverName}}, //[]*Ident      // field/method/parameter names; or nil if anonymous field
				Type:  f.ReceiverType,                                 //Expr          // field/method/parameter type
			},
		},
		}
	}
	fd = &ast.FuncDecl{
		Recv: receiver,
		Name: &ast.Ident{Name: f.Name},
		Type: &ast.FuncType{
			Params:  f.Params,
			Results: &ast.FieldList{List: []*ast.Field{}},
		},
		Body: &ast.BlockStmt{List: make([]ast.Stmt, 1)},
	}

	if f.ResultType != nil {
		fd.Type.Results.List = []*ast.Field{
			&ast.Field{
				Type: f.ResultType, //Expr          // field/method/parameter type
			},
		}
	}

	if f.Description != "" {
		fd.Doc = &ast.CommentGroup{
			List: []*ast.Comment{&ast.Comment{Text: "// " + f.Description}},
		}
	}
	// 	returnStmt,
	// }},

	//Preparing the Call("name", args)
	args := make([]ast.Expr, 1+f.Params.NumFields())
	args[0] = &ast.BasicLit{
		Kind:  token.STRING,
		Value: fmt.Sprintf("%q", f.JS),
	}

	for i, p := range f.Params.List {
		args[i+1] = p.Names[0]
	}

	ellipsis := token.NoPos //default is no ellipsis

	if f.Params.NumFields() > 0 {

		if _, ok := f.Params.List[f.Params.NumFields()-1].Type.(*ast.Ellipsis); ok {
			ellipsis = token.Pos(1) //it is an ellipsis create a position for the ellipsis
		}
	}
	selector := f.ReceiverName

	call := &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: &ast.Ident{
				Name: selector,
			},
			Sel: &ast.Ident{
				Name: "Call",
			},
		},
		Args:     args,
		Ellipsis: ellipsis,
	}

	//two cases: either we have to return something (and cast the call result) or not
	if f.ResultType == nil {
		fd.Body.List[0] = &ast.ExprStmt{X: call}
	} else {
		//I just need to build a reutrn statement and a conversion
		fd.Body.List[0] = &ast.ReturnStmt{
			Results: []ast.Expr{f.Convert(call)},
		}
	}
	return

}

func mConverter(e ast.Expr, method string) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X: e,
			Sel: &ast.Ident{
				Name: method,
			},
		},
	}
}

func IdentityConverter(e ast.Expr) ast.Expr  { return e }
func BoolConverter(e ast.Expr) ast.Expr      { return mConverter(e, "Bool") }
func StringConverter(e ast.Expr) ast.Expr    { return mConverter(e, "String") }
func IntConverter(e ast.Expr) ast.Expr       { return mConverter(e, "Int") }
func Int64Converter(e ast.Expr) ast.Expr     { return mConverter(e, "Int64") }
func Uint64Converter(e ast.Expr) ast.Expr    { return mConverter(e, "Uint64") }
func FloatConverter(e ast.Expr) ast.Expr     { return mConverter(e, "Float") }
func InterfaceConverter(e ast.Expr) ast.Expr { return mConverter(e, "Interface") }
