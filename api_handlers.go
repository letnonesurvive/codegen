package main

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

type ProfileResponse struct {
	Error string `json:"error"`
	User  *User  `json:"response,omitempty"`
}

type CreateResponse struct {
	Error string   `json:"error"`
	User  *NewUser `json:"response,omitempty"`
}

func ApiValidatorStructTags() map[string]struct{} {
	return map[string]struct{}{
		"required":  {},
		"paramname": {},
		"enum":      {},
		"default":   {},
		"min":       {},
		"max":       {},
	}
}

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

// query=login=mr.moderator&age=32&status=moderator&full_name=Ivan_Ivanov
// tag=`apivalidator:"enum=user|moderator|admin,default=user"`
func FillAndValidate(tag, fieldName string, field *reflect.Value, query url.Values) ApiError {

	if strings.Contains(tag, "paramname") {
		fieldName = ParseParams(tag)["paramname"]
	} else {
		fieldName = strings.ToLower(fieldName)
	}

	// need implement and use  setValue() function here
	queryValue := query[fieldName][0]

	field.SetString(queryValue)

	if strings.Contains(tag, "required") && len(queryValue) == 0 {
		return ApiError{Err: fmt.Errorf("%v must not be empty", fieldName)}
	}
	if strings.Contains(tag, "default") && len(queryValue) == 0 {
		defaulValue := ParseParams(tag)["default"]
		field.SetString(defaulValue)
	}

	if strings.Contains(tag, "enum") {
		tag, _ = strings.CutPrefix(tag, "enum=")
		enumValues := strings.Split(tag, "|")
		isValidEnum := false
		for _, v := range enumValues {
			if v == queryValue {
				isValidEnum = true
			}
		}
		if !isValidEnum {
			return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid enum value")}
		}
	}
	if strings.Contains(tag, "min") {
		min, _ := strconv.Atoi(ParseParams(tag)["min"])
		if len(queryValue) < min {
			return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid min")}
		}
	}
	if strings.Contains(tag, "max") {
		max, _ := strconv.Atoi(ParseParams(tag)["max"])
		if len(queryValue) > max {
			return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid max")}
		}
	}
	return ApiError{}
}

// func ValidateStruct(s interface{}) ApiError {
// 	srcStruct := reflect.ValueOf(s)
// 	if srcStruct.Kind() != reflect.Struct {
// 		return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("value not a struct")}
// 	}

// 	t := srcStruct.Type()
// 	for i := 0; i < srcStruct.NumField(); i++ {
// 		field := t.Field(i)
// 		fieldValue := srcStruct.Field(i)

// 		tags := strings.Split(field.Tag.Get("apivalidator"), ",")
// 		for _, tag := range tags {
// 			pair := strings.Split(tag, "=")
// 			if _, ok := ApiValidatorStructTags()[pair[0]]; !ok {
// 				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalig struct tag in field %s", tag)}
// 			}

// 			// err := ValidateTag(tag, fieldValue, strings.ToLower(field.Name))
// 			// if err.Err != nil {
// 			// 	return ApiError{HTTPStatus: err.HTTPStatus, Err: err}
// 			// }
// 		}
// 	}

// 	return ApiError{}
// }

func Decode(s interface{}, query url.Values) ApiError {
	srcValue := reflect.ValueOf(s)
	if srcValue.Elem().Kind() != reflect.Struct {
		return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("value not a struct")}
	}
	srcStruct := srcValue.Elem()
	t := srcStruct.Type()

	for i := 0; i < srcStruct.NumField(); i++ {
		field := t.Field(i)
		fieldValue := srcStruct.Field(i)
		FillAndValidate(field.Tag.Get("apivalidator"), field.Name, &fieldValue, query)

		// for _, tag := range tags {
		// 	var fieldName string
		// 	//var paramValue string

		// 	//pair := strings.Split(tag, "=")

		// if pair[0] == "paramname" {
		// 	fieldName = pair[1]
		// } else if _, ok := ApiValidatorStructTags()[pair[0]]; !ok {
		// 	return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid struct tag in field %s", pair[0])}
		// } else {
		// 	fieldName = strings.ToLower(field.Name)
		// }

		// //need to reimplement Validation. Call ValidateTag here? Do not define separate ValidateStruct
		// if queryValue, ok := query[fieldName]; ok {
		// 	paramValue = queryValue[0]
		// 	if pair[0] == "default" && paramValue == "" {
		// 		paramValue = pair[1]
		// 	}
		// 	if srcStruct.Field(i).CanSet() {
		// 		switch field.Type.Kind() {
		// 		case reflect.Int:
		// 			intValue, _ := strconv.Atoi(paramValue)
		// 			srcStruct.Field(i).SetInt(int64(intValue))
		// 		case reflect.String:
		// 			srcStruct.Field(i).SetString(paramValue)
		// 		}
		// 	}
		// } else {
		// 	//case 4 fail
		// 	return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("not found field name %s in query", fieldName)}
		// }
	}

	return ApiError{}
}

func (m *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/user/profile" { // насколько правильная идея делать сравнение именно так?
		var params ProfileParams
		if r.Method == http.MethodPost {
			bodyBytes, _ := io.ReadAll(r.Body)
			query, _ := url.ParseQuery(string(bodyBytes))
			Decode(&params, query)
			defer r.Body.Close()
		} else if r.Method == http.MethodGet {
			Decode(&params, r.URL.Query())
		}

		var response ProfileResponse

		if err := ValidateStruct(params); err.Err != nil {
			w.WriteHeader(err.HTTPStatus)
			response.Error = err.Error()
			data, _ := json.Marshal(response)
			w.Write(data)
			return
		}

		user, err := m.Profile(r.Context(), params)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response.Error = err.Error()
			data, _ := json.Marshal(response)
			w.Write(data)
			return
		}

		response.User = user
		data, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			response.Error = err.Error()
			data, _ := json.Marshal(response)
			w.Write(data)
		}
		w.Write(data)
	} else if r.URL.Path == "/user/create" {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusBadRequest)
		}
		var params CreateParams

		bodyBytes, _ := io.ReadAll(r.Body)
		query, _ := url.ParseQuery(string(bodyBytes))
		Decode(&params, query)
		defer r.Body.Close()

		if err := ValidateStruct(params); err.Err != nil {
			http.Error(w, err.Error(), err.HTTPStatus)
			return
		}

		user, err := m.Create(r.Context(), params)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		var response = CreateResponse{
			Error: "",
			User:  user,
		}

		data, _ := json.Marshal(response)
		w.Write(data)
	}
}
