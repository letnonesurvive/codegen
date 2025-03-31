package draft

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

func ParseParams(input string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(input, ",")

	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}

	return result
}

func SetValue(field *reflect.Value, fieldName string, query url.Values) error {

	queryValues, ok := query[fieldName]
	if !ok {
		return nil
	}
	if len(queryValues) > 1 {
		return ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("query value must be equal 1")}
	}

	switch field.Type().Kind() {
	case reflect.Int:
		intValue, err := strconv.Atoi(queryValues[0])
		if err != nil {
			return ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("%v must be %T", fieldName, int(field.Int()))}
		}
		field.SetInt(int64(intValue))
	case reflect.String:
		field.SetString(queryValues[0])
	}
	return nil
}

type ApiValidator struct {
}

func (a ApiValidator) apiValidatorStructTags() map[string]struct{} {
	return map[string]struct{}{
		"required":  {},
		"paramname": {},
		"enum":      {},
		"default":   {},
		"min":       {},
		"max":       {},
	}
}

// query=login=mr.moderator&age=32&status=moderator&full_name=Ivan_Ivanov
// tag=`apivalidator:"enum=user|moderator|admin,default=user"`
func (a ApiValidator) fillAndValidate(tag, fieldName string, field *reflect.Value, query url.Values) error {

	if strings.Contains(tag, "paramname") {
		fieldName = ParseParams(tag)["paramname"]
	} else {
		fieldName = strings.ToLower(fieldName)
	}

	err := SetValue(field, fieldName, query)
	if err != nil {
		return err
	}

	if strings.Contains(tag, "required") && field.IsZero() {
		return ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("%v must be not empty", fieldName)}
	}
	if strings.Contains(tag, "default") && field.IsZero() {
		defaulValue := ParseParams(tag)["default"]
		field.SetString(defaulValue)
	}

	if strings.Contains(tag, "enum") {
		tag, _ = strings.CutPrefix(tag, "enum=")
		tag, _, _ = strings.Cut(tag, ",")
		enumValues := strings.Split(tag, "|")
		isValidEnum := false
		for _, v := range enumValues {
			if v == field.String() {
				isValidEnum = true
			}
		}
		if !isValidEnum {
			return ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("%v must be one of [%v]", fieldName, strings.Join(enumValues, ", "))}
		}
	}
	if strings.Contains(tag, "min") {
		min, _ := strconv.Atoi(ParseParams(tag)["min"])
		switch field.Type().Kind() {
		case reflect.Int:
			if int(field.Int()) < min {
				return ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("%v must be >= %v", fieldName, min)}
			}
		case reflect.String:
			if len(field.String()) < min {
				return ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("%v len must be >= %v", fieldName, min)}
			}
		}
	}
	if strings.Contains(tag, "max") {
		max, _ := strconv.Atoi(ParseParams(tag)["max"])
		switch field.Type().Kind() {
		case reflect.Int:
			if int(field.Int()) > max {
				return ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("%v must be <= %v", fieldName, max)}
			}
		case reflect.String:
			if len(field.String()) > max {
				return ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("%v len must be <= %v", fieldName, max)}
			}
		}
	}
	return nil
}

func (a ApiValidator) Decode(s interface{}, query url.Values) error {
	srcValue := reflect.ValueOf(s)
	if srcValue.Elem().Kind() != reflect.Struct {
		return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("value not a struct")}
	}
	srcStruct := srcValue.Elem()
	t := srcStruct.Type()

	for i := 0; i < srcStruct.NumField(); i++ {
		field := t.Field(i)
		fieldValue := srcStruct.Field(i)
		err := a.fillAndValidate(field.Tag.Get("apivalidator"), field.Name, &fieldValue, query)
		if err != nil {
			return err
		}
	}

	return nil
}

type ProfileResponse struct {
	Error string `json:"error"`
	User  *User  `json:"response,omitempty"`
}

type CreateResponse struct {
	Error string   `json:"error"`
	User  *NewUser `json:"response,omitempty"`
}

func WriteError(w http.ResponseWriter, err error) {
	var response ProfileResponse
	if apiError, ok := err.(ApiError); ok {
		w.WriteHeader(apiError.HTTPStatus)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

	response.Error = err.Error()
	data, _ := json.Marshal(response)
	w.Write(data)
}

func (m *MyApi) handleProfile(w http.ResponseWriter, r *http.Request) {
	var params ProfileParams
	var validator ApiValidator
	if r.Method == http.MethodPost {
		bodyBytes, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		query, _ := url.ParseQuery(string(bodyBytes))
		err := validator.Decode(&params, query)
		if err != nil {
			WriteError(w, err)
			return
		}
	} else if r.Method == http.MethodGet {
		err := validator.Decode(&params, r.URL.Query())
		if err != nil {
			WriteError(w, err)
			return
		}
	}

	user, err := m.Profile(r.Context(), params)
	if err != nil {
		WriteError(w, err)
		return
	}

	var response ProfileResponse
	response.User = user
	data, err := json.Marshal(response)
	if err != nil {
		WriteError(w, ApiError{HTTPStatus: http.StatusInternalServerError, Err: err})
		return
	}

	w.Write(data)
}

func (m *MyApi) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, ApiError{HTTPStatus: http.StatusNotAcceptable, Err: fmt.Errorf("bad method")})
		return
	}
	auth, ok := r.Header["X-Auth"]
	if !ok || auth[0] != "100500" {
		WriteError(w, ApiError{HTTPStatus: http.StatusForbidden, Err: fmt.Errorf("unauthorized")})
		return
	}

	var params CreateParams

	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	query, _ := url.ParseQuery(string(bodyBytes))
	var validator ApiValidator
	err := validator.Decode(&params, query)

	if err != nil {
		WriteError(w, err)
		return
	}

	user, err := m.Create(r.Context(), params)
	if err != nil {
		WriteError(w, err)
		return
	}

	var response CreateResponse
	response.User = user
	data, _ := json.Marshal(response)
	w.Write(data)
}

func (m *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/user/profile":
		m.handleProfile(w, r)
	case "/user/create":
		m.handleCreate(w, r)
	default:
		WriteError(w, ApiError{HTTPStatus: http.StatusNotFound, Err: fmt.Errorf("unknown method")})
	}
}
