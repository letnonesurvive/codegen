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

func ValidateTag(tag string, value reflect.Value) error {

	if (tag == "required") && (value == reflect.Zero(value.Type())) {
		return fmt.Errorf("invalid value for required tag")
	} else if strings.Contains(tag, "enum") {
		values := strings.Split(tag, "|")
		for _, v := range values {
			if v != value.String() {
				return fmt.Errorf("invalid enum")
			}
		}
	} else if strings.Contains(tag, "min") {
		min, _ := strconv.Atoi(strings.Split(tag, "=")[1])
		switch value.Type().Kind() {
		case reflect.Int:
			if value.Int() < int64(min) {
				return fmt.Errorf("invalid min")
			}
		case reflect.String:
			if len(value.String()) < min {
				return fmt.Errorf("invalid min")
			}
		}
	} else if strings.Contains(tag, "max") {
		max, _ := strconv.Atoi(strings.Split(tag, "=")[1])
		switch value.Type().Kind() {
		case reflect.Int:
			if value.Int() > int64(max) {
				return fmt.Errorf("invalid min")
			}
		case reflect.String:
			if len(value.String()) > max {
				return fmt.Errorf("invalid min")
			}
		}
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

		tags := strings.Split(fieldType.Tag.Get("apivalidator"), ",") // 'paramname' pay attention
		for _, tag := range tags {
			pair := strings.Split(tag, "=")
			if _, ok := ApiValidatorStructTags()[pair[0]]; !ok {
				return fmt.Errorf("invalig struct tag in field %s", tag)
			}

			err := ValidateTag(tag, fieldValue)
			if err != nil {
				return fmt.Errorf("erorr with validation of tag %v with with field %s, %v", tag, fieldValue, err)
			}
		}
	}

	return nil
}

func Decode(s interface{}, query url.Values) error {
	srcValue := reflect.ValueOf(s)
	if srcValue.Elem().Kind() != reflect.Struct {
		return errors.New("value not a struct")
	}
	srcStruct := srcValue.Elem()
	t := srcStruct.Type()

	for i := 0; i < srcStruct.NumField(); i++ {
		fieldType := t.Field(i)
		tags := strings.Split(fieldType.Tag.Get("apivalidator"), ",") // 'paramname' pay attention
		for _, tag := range tags {
			var fieldName string
			pair := strings.Split(tag, "=")

			if pair[0] == "paramname" {
				fieldName = pair[1]
			} else if _, ok := ApiValidatorStructTags()[pair[0]]; !ok {
				return fmt.Errorf("invalig struct tag in field %s", pair[0])
			} else {
				fieldName = strings.ToLower(fieldType.Name)
			}

			if paramValue, ok := query[fieldName]; ok {
				if srcStruct.Field(i).CanSet() {
					switch fieldType.Type.Kind() {
					case reflect.Int:
						intValue, _ := strconv.Atoi(paramValue[0])
						srcStruct.Field(i).SetInt(int64(intValue))
					case reflect.String:
						srcStruct.Field(i).SetString(paramValue[0])
					}
				}
			} else {
				return fmt.Errorf("not found field name %s in query", fieldName)
			}
		}
	}

	return nil
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

		if err := ValidateStruct(params); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		user, err := m.Profile(r.Context(), params)
		if err != nil && err.Error() == "bad user" { // should use ApiError struct
			http.Error(w, err.Error(), 500)
			return
		}

		data, _ := json.Marshal(user)
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

		if err := ValidateStruct(params); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		user, err := m.Create(r.Context(), params)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		data, _ := json.Marshal(user)
		w.Write(data)
	}
}
