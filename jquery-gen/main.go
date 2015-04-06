// jquery-gen is a tool to generate gopherjs jquery's binding from api.jquery.com
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"go/format"
	"go/token"

	"github.com/ericaro/apigen"
	"github.com/ericaro/apigen/apijquery"
)

var (
	output = flag.String("o", "", "output directory (default to os.Stdout)")
	input  = flag.String("i", "", "input directory where are the entries.xml ")
)

func main() {
	flag.Parse()

	// create a writer either file (-o option) or stdout
	var target io.Writer
	if *output == "" {
		target = os.Stdout
	} else {
		file, err := os.Create(*output)
		if err != nil {
			panic(fmt.Errorf("cannot write to %v: %v", output, err))
		}
		defer file.Close()
		target = file
	}

	// parse each entry as described in th
	api, err := apijquery.Parse(*input)
	if err != nil {
		fmt.Printf("Error parsing xml entries: %v\n", err)
		os.Exit(-1)
	}

	c := apijquery.Compiler{}
	outapi, err := c.Compile(api)
	if err != nil {
		fmt.Printf("Compilation error: %v\n", err)
		os.Exit(-1)
	}

	err = format.Node(target, token.NewFileSet(), apigen.File(outapi))
	if err != nil {
		fmt.Printf("ast to .go error: %v\n", err)
		os.Exit(-1)
	}

	if *output == "" {
		fmt.Printf("generated %s/*.xml to stdout\n", *input)
	} else {
		fmt.Printf("generated %s/*.xml to %s\n", *input, *output)
	}
}
