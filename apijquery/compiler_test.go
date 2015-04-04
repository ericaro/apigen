package apijquery

import (
	"go/ast"
	"go/format"
	"go/token"
	"os"
)

func ExampleEntry() {
	//typical entry
	e := &Entry{
		Type:    "method",
		RawName: "jQuery.hide",
		Return:  "",
		Desc:    "comment",
		Signature: []Signature{
			Signature{
				Argument: []Argument{
					Argument{
						Name: "x",
						Type: "boolean",
					},
					Argument{
						Name: "v",
						Type: "Object",
					},
				},
			},
		},
	}
	f := &Entry{
		Type:    "property",
		RawName: "hide",
		Return:  "Object",
		Desc:    "comment",
	}

	fset := token.NewFileSet()
	fset.AddFile("api.go", -1, 100)
	out := File(
		//ImportDecl(ImportSpec("github.com/gopherjs/gopherjs/js"), ImportSpec("github.com/ericaro/sbr")),
		FuncDecl(e),
		GenDecl(token.TYPE, TypeSpec("QQuery", FieldGen(f))),
	)

	err := ast.Print(fset, out)
	err = format.Node(os.Stdout, fset, out)
	if err != nil {
		panic(err)
	}

	//Output: toto
}
