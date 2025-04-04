package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"text/template"
)

type ApiGen struct {
	Url    string `json:"url"`
	Auth   bool   `json:"auth"`
	Method string `json:"method"`
}

func findAllMethods(tree *ast.File) map[string][]ApiGen {
	res := make(map[string][]ApiGen)
	for _, decl := range tree.Decls {
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

func toCamel(s string) string {
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == ' ' || r == '-'
	})

	var res []string
	for _, word := range words {
		word = strings.Replace(word, string(word[0]), strings.ToUpper(string(word[0])), 1)
		res = append(res, word)
	}

	return strings.Join(res, "")
}

type tpl struct {
	MethodName string
}

var decodeStr = `err := validator.Decode(&params, query)
	if err != nil {
		WriteError(w, err)
		return
	}`

var validatorStr = `
	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	query, _ = url.ParseQuery(string(bodyBytes))
	` + decodeStr

var paramsTemplate = template.Must(template.New("paramsTemplate").Funcs(template.FuncMap{
	"camel": toCamel,
}).Parse(`
	var params {{.MethodName | camel}}Params
`))

func processPostRequest(out *os.File, funcName string) {
	// ошибки типа bad method и unauthorized должны браться из API?
	fmt.Fprintln(out, `	if r.Method != http.MethodPost {
		WriteError(w, ApiError{HTTPStatus: http.StatusNotAcceptable, Err: fmt.Errorf("bad method")})
		return
	}`)
	fmt.Fprint(out, `	auth, ok := r.Header["X-Auth"]
	if !ok || auth[0] != "100500" {
		WriteError(w, ApiError{HTTPStatus: http.StatusForbidden, Err: fmt.Errorf("unauthorized")})
		return
	}`)
	fmt.Fprintln(out, validatorStr)
}

func processGetRequest(out *os.File, funcName string) {
	fmt.Fprintf(out, `	if r.Method == http.MethodPost {
		bodyBytes, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		query, _ = url.ParseQuery(string(bodyBytes))
	} else if r.Method == http.MethodGet {
		query = r.URL.Query()
	}
	`)
	fmt.Fprintln(out, decodeStr)
}

func generateHTTPHandlers(methods map[string][]ApiGen, out *os.File) map[string]string {

	res := make(map[string]string)
	for structName, apis := range methods {
		for _, api := range apis {
			funcNames := strings.Split(api.Url, "/")
			funcName := funcNames[2] //funcNames[2] - name of source method which we use for create httphandle
			fmt.Fprintf(out, "func (s %v) handle%v (w http.ResponseWriter, r *http.Request) { \n", structName, funcName)
			paramsTemplate.Execute(out, tpl{funcName})
			fmt.Fprintf(out, "\tvar validator ApiValidator\n\tvar query url.Values\n")
			if api.Method == "POST" {
				processPostRequest(out, funcName)
			} else {
				processGetRequest(out, funcName)
			}
			fmt.Fprintln(out, "\n}")
			res[api.Url] = fmt.Sprintf("handle%v", funcName)
		}
	}
	return res
}

func generateServeHTTP(methods map[string][]ApiGen, handlers map[string]string, out *os.File) {

	type tpl struct {
		Url        string
		MethodName string
	}
	var template = template.Must(template.New("paramsTemplate").Parse(`	case "{{.Url}}":
	s.{{.MethodName}}(w, r)
`))

	for structName, apis := range methods {
		structArgumentName := "s"

		fmt.Fprintf(out, "func (%v *%v) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n\n", structArgumentName, structName)
		fmt.Fprintln(out, "\tw.Header().Set(\"Content-Type\", \"application/json\")")
		fmt.Fprintln(out, "\tswitch r.URL.Path {")
		for _, api := range apis {
			template.Execute(out, tpl{api.Url, handlers[api.Url]})
		}
		fmt.Fprintln(out, "\tdefault:\n\t\tWriteError(w, ApiError{HTTPStatus: http.StatusNotFound, Err: fmt.Errorf(\"unknown method\")})")
		fmt.Fprintln(out, "\t}\n}")
	}
}

func main() {

	if len(os.Args) < 3 {
		fmt.Println("Invalid arguments")
		return
	}

	fset := token.NewFileSet()
	tree, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	//ast.Fprint(os.Stdout, fset, tree, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	out, err := os.Create(os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Fprintln(out, "package "+tree.Name.Name)
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
	methods := findAllMethods(tree)

	//jsonErrorTag := strconv.Quote(`json:"error"`)
	fmt.Fprintf(out, "type ErrorResponse struct {\n \tError string `json:\"error\"` \n}\n")

	//help method to write error to body
	fmt.Fprintln(out, `func WriteError(w http.ResponseWriter, err error) {
	var response ErrorResponse
	if apiError, ok := err.(ApiError); ok {
		w.WriteHeader(apiError.HTTPStatus)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	response.Error = err.Error()
	data, _ := json.Marshal(response)
	w.Write(data)
}`)

	handlers := generateHTTPHandlers(methods, out)
	generateServeHTTP(methods, handlers, out)

	fmt.Fprintln(out) // empty string

	fmt.Println("Codegen complited")
}
