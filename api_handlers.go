package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
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
	value := reflect.ValueOf(s)
	if value.Kind() != reflect.Struct {
		return errors.New("value not a struct")
	}

	t := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := t.Field(i)
		fieldValue := value.Field(i)

		tag := fieldType.Tag.Get("apivalidator")
		if _, ok := StructTags()[tag]; !ok {
			return fmt.Errorf("invalig struct tag in field %s", tag)
		}

		err := ValidateTag(tag, fieldValue)
		if err != nil {
			return fmt.Errorf("erorr with validation of tag %v with with field %s, %v", tag, fieldValue, err)
		}
	}

	return nil
}

func (m *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/user/profile" { // на сколько правильная идея делать сравнение именно так
		var param ProfileParams
		param.Login = r.URL.Query().Get("login")
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
