package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

type ApiGen struct {
	Url    string `json:"url"`
	Auth   bool   `json:"auth"`
	Method string `json:"method"`
}

// func findAllStructs(node *ast.File) map[string]struct{} {
// 	res := make(map[string]struct{})
// 	for _, decl := range node.Decls {
// 		g, ok := decl.(*ast.GenDecl)
// 		if !ok {
// 			continue
// 		}
// 		for _, spec := range g.Specs {
// 			currType, ok := spec.(*ast.TypeSpec) // iterate over all types
// 			if !ok {
// 				continue
// 			}
// 			_, ok = currType.Type.(*ast.StructType)
// 			if !ok {
// 				continue
// 			}
// 			res[currType.Name.Name] = struct{}{}
// 		}
// 	}
// 	return res
// }

func findAllMethods(node *ast.File) map[string][]ApiGen {
	res := make(map[string][]ApiGen)
	for _, decl := range node.Decls {
		funcDecl, ok := decl.(*ast.FuncDecl)
		if !ok || funcDecl.Recv == nil {
			continue
		}

		if funcDecl.Doc == nil {
			continue
		}

		var typeName string
		recvType := funcDecl.Recv.List[0].Type
		switch t := recvType.(type) {
		case *ast.Ident:
			typeName = t.Name
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				typeName = ident.Name
			}
		}

		for _, comment := range funcDecl.Doc.List {
			var apiGen ApiGen
			if strings.HasPrefix(comment.Text, "// apigen:api") {
				after, _ := strings.CutPrefix(comment.Text, "// apigen:api")

				err := json.Unmarshal([]byte(after), &apiGen)
				if err != nil {
					fmt.Println(err)
				}
			}
			methods := res[typeName]
			res[typeName] = append(methods, apiGen)
		}
	}
	return res
}

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
	fmt.Fprintln(out, "import (")
	fmt.Fprintln(out, "\t\"encoding/json\"")
	fmt.Fprintln(out, "\t\"fmt\"")
	fmt.Fprintln(out, "\t\"io\"")
	fmt.Fprintln(out, "\t\"net/http\"")
	fmt.Fprintln(out, "\t\"net/url\"")
	fmt.Fprintln(out, "\t\"reflect\"")
	fmt.Fprintln(out, "\t\"strconv\"")
	fmt.Fprintln(out, "\t\"strings\"")
	fmt.Fprintln(out, ")")

	//structs := findAllStructs(node)
	methods := findAllMethods(node)

	for structName, apis := range methods {
		for _, api := range apis {
			//add camel style
			funcNames := strings.Split(api.Url, "/")
			funcName := strings.Join(funcNames, "")
			fmt.Fprintf(out, "func (s %v) handle%v (w http.ResponseWriter, r *http.Request) { \n\n}", structName, funcName)
			fmt.Fprintln(out)
		}
	}

	fmt.Fprintln(out) // empty string

	fmt.Println("Codegen complited")
}
