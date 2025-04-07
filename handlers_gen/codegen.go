package main

import (
	. "codegenhw/api_error"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"strings"
	"text/template"
)

type ApiGen struct {
	MethodName string
	ParamName  string
	ReturnName string
	Url        string `json:"url"`
	Auth       bool   `json:"auth"`
	Method     string `json:"method"`
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

		getTypeName := func(exp ast.Expr) string {
			var typeName string
			switch t := exp.(type) {
			case *ast.Ident:
				typeName = t.Name
			case *ast.StarExpr:
				if ident, ok := t.X.(*ast.Ident); ok {
					typeName = ident.Name
				}
			}
			return typeName
		}

		structName := getTypeName(funcDecl.Recv.List[0].Type)
		for _, comment := range funcDecl.Doc.List {
			var apiGen ApiGen
			if strings.HasPrefix(comment.Text, "// apigen:api") {
				after, _ := strings.CutPrefix(comment.Text, "// apigen:api")

				err := json.Unmarshal([]byte(after), &apiGen)
				if err != nil {
					fmt.Println(err)
				}
			}
			apiGen.MethodName = funcDecl.Name.Name
			apiGen.ParamName = getTypeName(funcDecl.Type.Params.List[1].Type)
			apiGen.ReturnName = getTypeName(funcDecl.Type.Results.List[0].Type)
			methods := res[structName]
			res[structName] = append(methods, apiGen)
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

type paramsTpl struct {
	ParamsName string
	StructName string
	MethodName string
}

var paramsTemplate = template.Must(template.New("paramsTemplate").Funcs(template.FuncMap{
	//"camel": toCamel,
}).Parse(`
	var params {{.ParamsName}}
	var response {{.StructName}}{{.MethodName}}Response
`))

type errorTpl struct {
	Condition string
	Error     error
}

func formatError(err error) string {
	if apiErr, ok := err.(ApiError); ok {
		return fmt.Sprintf("ApiError{HTTPStatus: %d, Err: fmt.Errorf(\"%s\")}", apiErr.HTTPStatus, apiErr.Err.Error())
	}
	return "err"
}

var errorTemplate = template.Must(template.New("errorTemplate").Funcs(template.FuncMap{
	"formatError": formatError,
}).Parse(`	if {{.Condition}} {
		WriteError(w, {{.Error | formatError}})
		return
}
`))

type responseTpl struct {
	StructName   string
	MethodName   string
	UserTypeName string
}

var responseTemplate = template.Must(template.New("responseTemplate").Parse("type {{.StructName}}{{.MethodName}}Response struct {\n" +
	"    Error string `json:\"error\"`\n" +
	"    User  *{{.UserTypeName}}  `json:\"response,omitempty\"`\n" +
	"}\n"))

func processPostRequest(out *os.File, funcName string) {
	// ошибки типа bad method и unauthorized должны браться из API?
	errorTemplate.Execute(out, errorTpl{Condition: "r.Method != http.MethodPost",
		Error: ApiError{HTTPStatus: http.StatusNotAcceptable, Err: fmt.Errorf("bad method")}})

	fmt.Fprintln(out, "auth, ok := r.Header[\"X-Auth\"]")

	errorTemplate.Execute(out, errorTpl{Condition: "!ok || auth[0] != \"100500\"",
		Error: ApiError{HTTPStatus: http.StatusForbidden, Err: fmt.Errorf("unauthorized")}})
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
			responseTemplate.Execute(out, responseTpl{structName, api.MethodName, api.ReturnName})
			fmt.Fprintf(out, "func (s %v) handle%v (w http.ResponseWriter, r *http.Request) { \n", structName, api.MethodName)
			paramsTemplate.Execute(out, paramsTpl{api.ParamName, structName, api.MethodName})
			//apivalidator быть не должен, должен быть метод unpack свой для каждой структуры, его также нужно генерировать.
			fmt.Fprintf(out, "\tvar validator ApiValidator\n\tvar query url.Values\n")
			if api.Method == "POST" {
				processPostRequest(out, api.MethodName)
			} else {
				processGetRequest(out, api.MethodName)
			}
			fmt.Fprintf(out, "\tuser, err := s.%v(r.Context(), params)\n", toCamel(api.MethodName))
			errorTemplate.Execute(out, errorTpl{Condition: "err != nil", Error: fmt.Errorf("")})
			fmt.Fprintln(out, "response.User = user")
			fmt.Fprintln(out, "data, err := json.Marshal(response)")
			errorTemplate.Execute(out, errorTpl{Condition: "err != nil",
				Error: ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("err")}}) // недоработка вот здесь
			fmt.Fprintln(out, "w.Write(data)")
			fmt.Fprintln(out, "\n}")
			res[api.Url] = fmt.Sprintf("handle%v", api.MethodName)
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
		fmt.Fprintf(out, "\tdefault:\n\t\t WriteError(w,%v)", formatError(ApiError{HTTPStatus: http.StatusNotFound, Err: fmt.Errorf("unknown method")}))
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
	fmt.Fprintln(out, ". \"codegenhw/api_error\"")
	fmt.Fprintln(out, "\t\"fmt\"")
	fmt.Fprintln(out, "\t\"io\"")
	fmt.Fprintln(out, "\t\"net/http\"")
	fmt.Fprintln(out, "\t\"net/url\"")
	fmt.Fprintln(out, ")")

	//structs := findAllStructs(node)
	methods := findAllMethods(tree)

	fmt.Fprintf(out, "type ErrorResponse struct {\n \tError string `json:\"error\"` \n}\n")

	//writes help method to send error in body
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
