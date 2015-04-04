package apigen

import (
	"go/ast"
	"go/printer"
	"go/token"
	"os"
)

func ExampleField() {

	p := &Property{
		Name: "Toto",
		Type: JSObject,
		JS:   "toto",
	}

	f := Field(p)
	//ast.Print(token.NewFileSet(), f)

	//building a "decorative" struct definition otherwise Fprint won't print anything !
	t := &ast.StructType{
		Fields: &ast.FieldList{List: []*ast.Field{f}},
	}

	printer.Fprint(os.Stdout, token.NewFileSet(), t)
	//Output:
	// struct {
	//	Toto *js.Object `js:"toto"`
	// }
}

func ExampleTypeDecl() {

	ty := &Type{
		Name: "Foo",
		Properties: []*Property{
			&Property{
				Name: "Bar",
				Type: JSObject,
				JS:   "bar",
			},
			&Property{
				Name: "Baz",
				Type: &ast.Ident{Name: "bool"},
				JS:   "baz",
			},
		},
	}

	td := TypeDecl(ty)
	printer.Fprint(os.Stdout, token.NewFileSet(), td)
	//Output:
	// type Foo struct {
	// 	*js.Object
	// 	Bar	*js.Object	`js:"bar"`
	// 	Baz	bool		`js:"baz"`
	// }

}
func ExampleCtor() {
	ty := &Type{
		Name: "Foo",
	}
	td := Ctor(ty)
	printer.Fprint(os.Stdout, token.NewFileSet(), td)
	//Output:
	// func newFoo(j *js.Object) Foo {
	// 	return Foo{Object: j}
	// }
}

func ExampleFuncDecl() {

	f := &Func{
		//Description: "Foo function",
		Name:         "Foo",
		JS:           "foo",
		ReceiverName: "JQ",
		Params: &ast.FieldList{
			List: []*ast.Field{
				&ast.Field{
					Names: []*ast.Ident{&ast.Ident{Name: "b"}},
					Type:  &ast.Ident{Name: "bool"},
				},
			}},
		ResultType: &ast.Ident{Name: "bool"},
		Convert:    BoolConverter,
	}

	td := FuncDecl(f)
	printer.Fprint(os.Stdout, token.NewFileSet(), td)
	//Output:
	// func Foo(b bool) bool {
	// 	return JQ.Call("foo", b).Bool()
	// }
}

func ExampleImportDecl() {
	impDecl := ImportDecl([]string{"github.com/gopherjs/gopherjs/js", "github.com/gopherjs/gopherjs/jquery"})

	printer.Fprint(os.Stdout, token.NewFileSet(), impDecl)
	//Output:
	// import (
	// 	"github.com/gopherjs/gopherjs/js"
	// 	"github.com/gopherjs/gopherjs/jquery"
	// )

}

func ExampleFile() {

	// a simple test api
	api := &Api{
		Name:    "jquery",
		Imports: []string{"github.com/gopherjs/gopherjs/js", "github.com/gopherjs/gopherjs/jquery"},
		Types: []*Type{
			&Type{
				Name: "Foo",
				Properties: []*Property{
					&Property{
						Name: "Bar",
						Type: JSObject,
						JS:   "bar",
					},
					&Property{
						Name: "Baz",
						Type: &ast.Ident{Name: "bool"},
						JS:   "baz",
					},
				},
			}},
		Funcs: []*Func{
			&Func{
				//Description: "Foo function",
				Name: "Fooer",
				JS:   "foo",
				Params: &ast.FieldList{
					List: []*ast.Field{
						&ast.Field{
							Names: []*ast.Ident{&ast.Ident{Name: "b"}},
							Type:  &ast.Ident{Name: "bool"},
						},
					}},
				ResultType: &ast.Ident{Name: "bool"},
				Convert:    BoolConverter,
			}},
	}

	file := File(api)
	printer.Fprint(os.Stdout, token.NewFileSet(), file)
	//Output:
	// package jquery

	// import (
	// 	"github.com/gopherjs/gopherjs/js"
	// 	"github.com/gopherjs/gopherjs/jquery"
	// )

	// type Foo struct {
	// 	*js.Object
	// 	Bar	*js.Object	`js:"bar"`
	// 	Baz	bool		`js:"baz"`
	// }

	// func newFoo(j *js.Object) Foo {
	// 	return Foo{Object: j}
	// }
	// func Fooer(b bool) (x bool) {
	// 	return JQ.Call("foo", b).Bool()
	// }

}
