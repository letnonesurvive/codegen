package main

import (
	"fmt"
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

func SetValue(field *reflect.Value, fieldName string, query url.Values) {

	queryValues, ok := query[fieldName]
	if !ok || len(queryValues) > 1 {
		return
	}

	switch field.Type().Kind() {
	case reflect.Int:
		intValue, _ := strconv.Atoi(queryValues[0])
		field.SetInt(int64(intValue))
	case reflect.String:
		field.SetString(queryValues[0])
	}
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
func (a ApiValidator) fillAndValidate(tag, fieldName string, field *reflect.Value, query url.Values) ApiError {

	if strings.Contains(tag, "paramname") {
		fieldName = ParseParams(tag)["paramname"]
	} else {
		fieldName = strings.ToLower(fieldName)
	}

	SetValue(field, fieldName, query)

	if strings.Contains(tag, "required") && field.IsZero() {
		return ApiError{HTTPStatus: http.StatusBadRequest, Err: fmt.Errorf("%v must be not empty", fieldName)}
	}
	if strings.Contains(tag, "default") && field.IsZero() {
		defaulValue := ParseParams(tag)["default"]
		field.SetString(defaulValue)
	}

	if strings.Contains(tag, "enum") {
		tag, _ = strings.CutPrefix(tag, "enum=")
		enumValues := strings.Split(tag, "|")
		isValidEnum := false
		for _, v := range enumValues {
			if v == field.String() {
				isValidEnum = true
			}
		}
		if !isValidEnum {
			return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid enum value")}
		}
	}
	if strings.Contains(tag, "min") {
		min, _ := strconv.Atoi(ParseParams(tag)["min"])
		switch field.Type().Kind() {
		case reflect.Int:
			if int(field.Int()) < min {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid min")}
			}
		case reflect.String:
			if len(field.String()) < min {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid min")}
			}
		}
	}
	if strings.Contains(tag, "max") {
		max, _ := strconv.Atoi(ParseParams(tag)["max"])
		switch field.Type().Kind() {
		case reflect.Int:
			if int(field.Int()) > max {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid max")}
			}
		case reflect.String:
			if len(field.String()) > max {
				return ApiError{HTTPStatus: http.StatusInternalServerError, Err: fmt.Errorf("invalid max")}
			}
		}
	}
	return ApiError{}
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
		if err.Err != nil {
			return ApiError{HTTPStatus: err.HTTPStatus, Err: err.Err}
		}
	}

	return nil
}
