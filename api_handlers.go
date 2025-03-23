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
	User  User   `json:"response"`
}

type CreateResponse struct {
	Error string  `json:"error"`
	User  NewUser `json:"response"`
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

func ValidateTag(tag string, value reflect.Value) ApiError {

	if (tag == "required") && (value == reflect.Zero(value.Type())) {
		return ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("invalid value for required tag")}
	} else if strings.Contains(tag, "enum") {
		tag, _ = strings.CutPrefix(tag, "enum=")
		enumValues := strings.Split(tag, "|")
		for _, v := range enumValues {
			if v == value.String() {
				return ApiError{}
			}
		}
		return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid enum value")}
	} else if strings.Contains(tag, "min") {
		min, _ := strconv.Atoi(strings.Split(tag, "=")[1])
		switch value.Type().Kind() {
		case reflect.Int:
			if value.Int() < int64(min) {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid min")}
			}
		case reflect.String:
			if len(value.String()) < min {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid min")}
			}
		}
	} else if strings.Contains(tag, "max") {
		max, _ := strconv.Atoi(strings.Split(tag, "=")[1])
		switch value.Type().Kind() {
		case reflect.Int:
			if value.Int() > int64(max) {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid min")}
			}
		case reflect.String:
			if len(value.String()) > max {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid min")}
			}
		}
	} else if strings.Contains(tag, "default") && (value == reflect.Zero(value.Type())) {
		return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid default")}
	}
	return ApiError{}
}

func ValidateStruct(s interface{}) ApiError {
	srcStruct := reflect.ValueOf(s)
	if srcStruct.Kind() != reflect.Struct {
		return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("value not a struct")}
	}

	t := srcStruct.Type()
	for i := 0; i < srcStruct.NumField(); i++ {
		field := t.Field(i)
		fieldValue := srcStruct.Field(i)

		tags := strings.Split(field.Tag.Get("apivalidator"), ",") // 'paramname' pay attention
		for _, tag := range tags {
			pair := strings.Split(tag, "=")
			if _, ok := ApiValidatorStructTags()[pair[0]]; !ok {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalig struct tag in field %s", tag)}
			}

			err := ValidateTag(tag, fieldValue)
			if err.Err != nil {
				return ApiError{HTTPStatus: err.HTTPStatus, Err: fmt.Errorf("error with validation of tag %v with with field %s, %v", tag, fieldValue, err)}
			}
		}
	}

	return ApiError{}
}

func Decode(s interface{}, query url.Values) ApiError {
	srcValue := reflect.ValueOf(s)
	if srcValue.Elem().Kind() != reflect.Struct {
		return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("value not a struct")}
	}
	srcStruct := srcValue.Elem()
	t := srcStruct.Type()

	for i := 0; i < srcStruct.NumField(); i++ {
		field := t.Field(i)
		tags := strings.Split(field.Tag.Get("apivalidator"), ",")
		for _, tag := range tags {
			var fieldName string
			var paramValue string
			pair := strings.Split(tag, "=")

			if pair[0] == "paramname" {
				fieldName = pair[1]
			} else if _, ok := ApiValidatorStructTags()[pair[0]]; !ok {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalig struct tag in field %s", pair[0])}
			} else {
				fieldName = strings.ToLower(field.Name)
			}

			if queryValue, ok := query[fieldName]; ok {
				paramValue = queryValue[0]
				if pair[0] == "default" && paramValue == "" {
					paramValue = pair[1]
				}
				if srcStruct.Field(i).CanSet() {
					switch field.Type.Kind() {
					case reflect.Int:
						intValue, _ := strconv.Atoi(paramValue)
						srcStruct.Field(i).SetInt(int64(intValue))
					case reflect.String:
						srcStruct.Field(i).SetString(paramValue)
					}
				}
			} else {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("not found field name %s in query", fieldName)}
			}
		}
	}

	return ApiError{}
}

func (m *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/user/profile" { // насколько правильная идея делать сравнение именно так
		var params ProfileParams
		if r.Method == http.MethodPost {
			bodyBytes, _ := io.ReadAll(r.Body)
			query, _ := url.ParseQuery(string(bodyBytes))
			Decode(&params, query)
			defer r.Body.Close()
		} else if r.Method == http.MethodGet {
			Decode(&params, r.URL.Query())
		}

		if err := ValidateStruct(params); err.Err != nil {
			http.Error(w, err.Error(), err.HTTPStatus)
			return
		}

		user, err := m.Profile(r.Context(), params)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var response = ProfileResponse{
			Error: "",
			User:  *user,
		}

		data, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "error marshal json", 500)
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
			User:  *user,
		}

		data, _ := json.Marshal(response)
		w.Write(data)
	}
}
