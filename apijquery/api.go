package apijquery

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

//Api represent the api.jquery.com raw parsing
//
// the api is made of single entry files (Entry)
// and <entries> with several entry inside (mainly for "conflic" reasons)
type Api struct {
	Entries []*Entries
	Entry   []*Entry
}

//NewApi creates a newly allocated empty API
func NewApi() *Api {
	return &Api{
		Entry:   make([]*Entry, 0),
		Entries: make([]*Entries, 0),
	}
}

//Entries is to read some of the xml files
// instead of having an <entry> element they have
// <entries> <desc> and <entry>*
// so to read "entries" I need a union type entryOrDesc, this is it
type Entries struct {
	Desc  string   `xml:"desc"`
	Entry []*Entry `xml:"entry"`
}

//Argument describe a signature part
type Argument struct {
	Name       string `xml:"name,attr"`
	Type       string `xml:"type,attr"`
	Desc       string `xml:"desc"`
	Optional   bool   `xml:"optional,attr"`
	Deprecated string `xml:"deprecated,attr"`
	Removed    string `xml:"removed,attr"`
}

//Signature one of many possible signature for a single function
type Signature struct {
	Added    string     `xml:"added"`
	Argument []Argument `xml:"argument"`
	Variadic bool
}

//Entry the principal item in the jquery api
type Entry struct {
	Type       string      `xml:"type,attr"`
	Deprecated string      `xml:"deprecated,attr"`
	Removed    string      `xml:"removed,attr"`
	RawName    string      `xml:"name,attr"`
	Return     string      `xml:"return,attr"`
	Title      string      `xml:"entry>title"`
	Desc       string      `xml:"desc"`
	Signature  []Signature `xml:"signature"`
}

//Receiver computes the expected receiver
//
// jquery has the following convention
// Deferred.foo for a foo method on the Deferred Object
//
// But it has the following exception:
//
// foo -> receiver JQuery
//
// JQuery.foo -> receiver none (static function)
//
// Nevertheless, we return jQuery for jQuery.foo, and "" for "add"
func (e Entry) Receiver() string {

	parts := strings.Split(e.RawName, ".")
	switch {
	case len(parts) > 1:
		return strings.Join(parts[0:len(parts)-1], ".")
	default:
		return ""
	}
}

//ReturnVoid true if there is no "return" type
func (e Entry) ReturnVoid() bool { return e.Return == "" || e.Return == "undefined" }

//Name return the method name part (cleaned up of all the prefix stuff)
func (e Entry) Name() string {
	parts := strings.Split(e.RawName, ".")
	return parts[len(parts)-1]
}

//Title is a function to uppercase the first letter of a name
func Title(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}
