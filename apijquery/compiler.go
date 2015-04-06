package apijquery

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"go/ast"

	"github.com/ericaro/apigen"
)

//Compiler convert an *Api into the *apigen.Api intermediate object.
//
// *apigen.Api is a struct that can be generated into go source file.
//
//
type Compiler struct{}

//isOk return true if I have to keep the entry
func (c Compiler) isOk(p *Entry) bool {
	return p.Deprecated == "" && p.Removed == ""
}

//logRejected just print out info about a rejected entry
func (c Compiler) logRejected(p *Entry) {
	if p.Removed != "" {
		log.Printf("Skipping %v : removed=%v deprecated=%v", p.RawName, p.Removed, p.Deprecated)
	} else {
		log.Printf("Skipping %v : deprecated=%v", p.RawName, p.Deprecated)
	}
}

//Compile the current jquery api into the independent apigen one
func (c Compiler) Compile(api *Api) (out *apigen.Api, err error) {

	out = &apigen.Api{
		Name:    "jquery",
		Imports: []string{"github.com/gopherjs/gopherjs/js"},
	}

	//first collect all
	all := make([]*Entry, 0, len(api.Entry)+2*len(api.Entries))

	for _, p := range api.Entry {
		if c.isOk(p) {
			all = append(all, p)
		} else {
			c.logRejected(p)
		}
	}

	for _, e := range api.Entries {
		for _, p := range e.Entry {
			if c.isOk(p) {
				all = append(all, p)
			} else {
				c.logRejected(p)
			}
		}
	}

	// Now all "OK" entries are in "all" (getting rid of deprecated, and removed ones)

	//collect all types defined in the API, into a set of declared receivers
	typenames := make(map[string]interface{}) // map of all types found
	for _, e := range all {
		typenames[e.Receiver()] = nil // this identify the type
	}

	//Deal with EXCEPTIONS

	// remove some types that are not supported
	delete(typenames, "jQuery.browser")
	delete(typenames, "jQuery.fx")
	delete(typenames, "jQuery.fn")

	for _, e := range all {
		switch {
		// rename Callbacks it was in fact a constructor, and it's name will collide with the Callbacks definition type
		case e.RawName == "jQuery.Callbacks":
			e.RawName = "jQuery.newCallbacks"

			// rename Deferred it was in fact a constructor, and it's name will collide with the Deferred definition type
		case e.RawName == "jQuery.Deferred":
			e.RawName = "jQuery.newDeferred"

		//move jquery.fn methods (only one right now) as prefixed Fn directly to jQuery
		case strings.HasPrefix(e.RawName, "jQuery.fn"):
			e.RawName = "jQuery.Fn" + Title(e.Name())
		}
	}

	// entries need to be sorted by name so the api generation has no "random" order

	//sort types by name
	names := make([]string, 0, len(typenames))
	for name := range typenames {
		names = append(names, name)
	}
	sort.Strings(names)

	//for each type create a corresponding apigen type
	for _, tyname := range names {
		log.Printf("processing %v", tyname)

		// keep methods and properties into a map (for key unicity)
		// we'll sort the map later
		methods := make(map[string]*Entry)
		properties := make(map[string]*Entry)

		for _, e := range all {
			switch {
			case e.Receiver() != tyname:
				//nothing to do, this is the entry for another type
			case e.Type == "method":
				if was, exists := methods[e.Name()]; exists {
					//check that an entry with the same name not already exists (this is possible)
					x := merge(was, e) // inplace merge
					// merge has the "permission" to change the entry name, before storing it
					methods[x.Name()] = x
				} else {
					methods[e.Name()] = e
				}
				// in any case, entry can have "multiple" signature for the same function.
				// in go we do not have this, so we need to fallback to the most generic interface (...interface{})
				mergeSignatures(methods[e.Name()])

			case e.Type == "property":
				if _, exists := properties[e.Name()]; exists {
					panic(fmt.Errorf("duplicated property"))
				}
				properties[e.Name()] = e
			}
		}
		// now I've got all the methods, properties for a given type
		//compute their  go type name
		var gotypename string
		switch tyname {
		case "":
			gotypename = "JQuery"

		case "jQuery":
			gotypename = ""

		//unsupported objects
		case "event", "callbacks", "deferred": //supported objects
			gotypename = Title(tyname)
		}

		// and build the correct apigen.Type.
		// the "" is a special "type" to generate functions

		var ty *apigen.Type
		if gotypename != "" { // this is a regular type

			ty = &apigen.Type{
				Name:       gotypename,
				Properties: make([]*apigen.Property, 0, len(properties)),
			}
			log.Printf("Compiling Type %v", gotypename)
			out.Types = append(out.Types, ty)

			//sort by name and compile properties
			names := make([]string, 0, len(properties))
			for k := range properties {
				names = append(names, k)
			}
			sort.Strings(names)

			for _, n := range names {
				e := properties[n]
				ty.Properties = append(ty.Properties, &apigen.Property{
					Name: Title(e.Name()),  //string
					JS:   e.Name(),         //string   // name in js
					Type: goType(e.Return), //ast.Expr //expression defining a type
				})

			}
		}

		//sort by name and compile funcs
		names := make([]string, 0, len(methods))
		for k := range methods {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, n := range names {

			// there is the special case of gotypename == "" that represent a static
			// call to jquery
			var rname string   //receiver variable name
			var rtype ast.Expr // receiver type (nil for the special case )

			if gotypename == "" { // static function
				rname, rtype = "JQ", nil
			} else {
				rname, rtype = "x", &ast.Ident{Name: gotypename}
			}
			//everything else is straightforward
			e := methods[n]
			out.Funcs = append(out.Funcs, &apigen.Func{
				Description:  e.Desc,
				ReceiverType: rtype,
				ReceiverName: rname,
				Name:         Title(e.Name()),               //    string
				JS:           e.Name(),                      //    string
				ResultType:   goType(e.Return),              //Expr          // field/method/parameter type
				Params:       compileParams(e.Signature[0]), //    *ast.FieldList
				Convert:      converterFor(e.Return),        //    func(ast.Expr) ast.Expr //the expression that deals with types
			})
		}

	}
	return
}

//compileParams returns the field list from a given signature
func compileParams(s Signature) *ast.FieldList {
	fields := make([]*ast.Field, 0, 10)

	for i, a := range s.Argument {
		ty := goType(a.Type)
		if ty == nil {
			panic(fmt.Errorf("invalid api: unknown type %v", a.Type))
		}
		if s.Variadic && i == len(s.Argument)-1 { //last and variadic
			ty = &ast.Ellipsis{
				Elt: ty,
			}
		}
		fields = append(fields, &ast.Field{
			Names: []*ast.Ident{
				&ast.Ident{Name: escapeReservedWord(a.Name)},
			},
			Type: ty,
		})
	}
	return &ast.FieldList{
		List: fields,
	}
}

//converterFor return the correct "Convert" for the given return type.
//
// a converter is a function that converts an ast.Expr (which value has type *js.Object) into
// another expression which value must have the type 'name'
//
// 'name' is the name found in the jquery documentation
//
//apigen has a bunch of ready to use converter, we just need to map them according
// to the convention in jquery doc.
func converterFor(name string) func(ast.Expr) ast.Expr {

	switch name {

	case "Boolean", "boolean":
		return apigen.BoolConverter

	case "Number":
		return apigen.FloatConverter

	case "Integer":
		return apigen.IntConverter

	case "String", "selector", "Selector":
		return apigen.StringConverter

	case "undefined", "":
		return apigen.InterfaceConverter

	//unsupported
	case "Object", "jqXHR", "Function", "Promise", "Array", "XMLDocument", "Element":
		return apigen.IdentityConverter

	case "jQuery", "Event", "Callbacks", "Deferred": //supported objects

		return func(j ast.Expr) ast.Expr {
			return &ast.CallExpr{
				Fun: &ast.Ident{
					Name: fmt.Sprintf("new%s", Title(name)),
				},
				Args: []ast.Expr{j},
				//Ellipsis: token.Pos(120),
			}
		}

	default:
		panic(fmt.Errorf("unknown type %s", name))
	}
}

//merge two method entries.
//
// panic in impossible cases
func merge(o, n *Entry) *Entry {
	if o.Type != n.Type {
		panic(fmt.Errorf("type mismatch for entry merge: %v vs %v", o.Type, n.Type))
	}
	if o.RawName != n.RawName {
		panic(fmt.Errorf("raw name mismatch for entry merge: %v vs %v", o.RawName, n.RawName))
	}

	//deal with return type
	var mreturn string
	if o.Return == n.Return {
		mreturn = o.Return
	} else {
		log.Printf("merging return type %v <> %v", o.Return, n.Return)
		mreturn = "Object" // by default
	}

	return &Entry{
		Type:      o.Type, // they must have the same type
		RawName:   o.RawName,
		Return:    mreturn,
		Desc:      o.Desc + "\n// OR\n// " + n.Desc,
		Signature: append(append(make([]Signature, 0, 10), o.Signature...), n.Signature...),
	}

}

func mergeSignatures(x *Entry) {
	//s is the post generic signature
	s := Signature{
		Argument: []Argument{
			Argument{
				Name: "i",
				Type: "interface{}",
			},
		},
		Variadic: true,
	}

	//TODO: sometimes you don't need to be that violent
	if len(x.Signature) > 1 {
		x.Signature = []Signature{s}
	}

	// if an argument is optional it need to be a generic variadic interface
	for _, a := range x.Signature[0].Argument {
		if a.Optional {
			x.Signature = []Signature{s}
			break
		}
	}

}

var reservedWords = map[string]interface{}{
	"break":     nil,
	"default":   nil,
	"func":      nil,
	"interface": nil,
	"select":    nil,
	"case":      nil,
	"defer":     nil,
	"go":        nil,
	"map":       nil,
	"struct":    nil,
	"chan":      nil,
	"else":      nil,
	"goto":      nil,
	"package":   nil,
	"switch":    nil,
	"const":     nil,
	"if":        nil,
	"range":     nil,
	"type":      nil,
	"continue":  nil,
	"for":       nil,
	"import":    nil,
	"return":    nil,
	"var":       nil,
	// in the func generation we introduce two new "word" that must be reserved
	// func (x XXX) method(var){
	//    j= x.Call()
	//    return j
	// }
	// thereofre x, and j are forbidden as "var" values
	"j": nil,
	"x": nil,
}

func escapeReservedWord(word string) string {
	if _, exists := reservedWords[word]; exists {
		return Title(word)
	}
	return word
}

//goType return the ast.Expr defining the golang type for the jquery declared type
func goType(s string) (t ast.Expr) {
	// defer func() {
	// 	log.Printf("generated type Exp %s -> %v", s, FormatNode(t))
	// }()
	//s is the real receiver described in the entry file
	// "" for JQuery
	// jQuery for nil type etc
	switch s {
	case "", "undefined", "interface{}":
		return &ast.InterfaceType{
			Methods: &ast.FieldList{},
		}
	case "jQuery":
		return &ast.Ident{Name: "JQuery"}

	case "Boolean", "boolean":
		return &ast.Ident{Name: "bool"}

	case "Number":
		return &ast.Ident{Name: "float64"}

	case "Integer":
		return &ast.Ident{Name: "int"}

	case "String", "selector", "Selector":
		return &ast.Ident{Name: "string"}

	//unsupported objects
	case "event", "callbacks", "deferred", "Event", "Callbacks", "Deferred": //supported objects
		return &ast.Ident{Name: Title(s)}

	default:
		//case "Object", "jqXHR", "Function", "Promise", "Array", "XMLDocument", "Element", "PlainObject", "Anything":
		return &ast.StarExpr{
			X: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "js"},
				Sel: &ast.Ident{Name: "Object"},
			},
		}
		//	panic(fmt.Errorf("unknown type %s", s))
	}
}
