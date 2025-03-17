package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

func main() {

	if len(os.Args) < 3 {
		fmt.Println("Invalid arguments")
		return
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		fmt.Println(err)
		return
	}

	out, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Fprintln(out, "package "+node.Name.Name)
	fmt.Fprintln(out, "import "+"\"net/http\"")

	for _, decl := range node.Decls {
		g, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range g.Specs {
			currType, ok := spec.(*ast.TypeSpec) // iterate over all types
			if !ok {
				fmt.Printf("SKIP %#T is not ast.TypeSpec\n", spec)
				continue
			}
			fmt.Println(currType)
		}
	}

	fmt.Fprintln(out) // empty string

	fmt.Println("Codegen complited")
}
