package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

func StructTags() map[string]struct{} {
	return map[string]struct{}{
		"required":  {},
		"paramname": {},
		"enum":      {},
		"default":   {},
		"min":       {},
		"max":       {},
	}
}

func ValidateTag(tag string, value reflect.Value) error {

	if (tag == "required") && (value == reflect.Zero(value.Type())) {
		return fmt.Errorf("invalid required tag")
	}
	return nil
}

func ValidateStruct(s interface{}) error {
	srcStruct := reflect.ValueOf(s)
	if srcStruct.Kind() != reflect.Struct {
		return errors.New("value not a struct")
	}

	t := srcStruct.Type()
	for i := 0; i < srcStruct.NumField(); i++ {
		fieldType := t.Field(i)
		fieldValue := srcStruct.Field(i)

		tag := fieldType.Tag.Get("apivalidator")
		if _, ok := StructTags()[tag]; tag == "" || !ok {
			return fmt.Errorf("invalig struct tag in field %s", tag)
		}

		err := ValidateTag(tag, fieldValue)
		if err != nil {
			return fmt.Errorf("erorr with validation of tag %v with with field %s, %v", tag, fieldValue, err)
		}
	}

	return nil
}

func Decode(s interface{}, query string) error {
	srcValue := reflect.ValueOf(s)
	if srcValue.Elem().Kind() != reflect.Struct {
		return errors.New("value not a struct")
	}

	params, _ := url.ParseQuery(query)

	srcStruct := srcValue.Elem()
	t := srcStruct.Type()
	for i := 0; i < srcStruct.NumField(); i++ {
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("apivalidator") // 'paramname' pay attention
		if _, ok := StructTags()[tag]; tag == "" || !ok {
			return fmt.Errorf("invalig struct tag in field %s", tag)
		}
		fieldName := strings.ToLower(fieldType.Name)
		if paramValue, ok := params[fieldName]; ok {
			if srcStruct.Field(i).CanSet() {
				switch reflect.ValueOf(fieldName).Kind() {
				case reflect.Int:
					intValue, _ := strconv.Atoi(paramValue[0])
					srcStruct.Field(i).SetInt(int64(intValue))
				case reflect.String:
					srcStruct.Field(i).SetString(paramValue[0])
				}
			}
		}
	}

	return nil
}

func (m *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/user/profile" { // на сколько правильная идея делать сравнение именно так
		var param ProfileParams
		if r.Method == http.MethodPost {
			bodyBytes, _ := io.ReadAll(r.Body)
			Decode(&param, string(bodyBytes))
			defer r.Body.Close()
		}
		if err := ValidateStruct(param); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		user, err := m.Profile(r.Context(), param)
		if err != nil && err.Error() == "bad user" {
			http.Error(w, err.Error(), 500)
			return
		}
		data, _ := json.Marshal(user)
		w.Write(data)
	} else if r.URL.Path == "/user/create" {
		if r.Method != http.MethodPost {
			return
		}
		var param CreateParams
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&param) // need to decode myself
		if err != nil {
			http.Error(w, "error to unmarhsll of body", 500)
		}

		if err := ValidateStruct(param); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		m.Create(r.Context(), param)

	}
}
