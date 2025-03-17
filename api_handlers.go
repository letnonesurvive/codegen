package main

import (
	"net/http"
	"strconv"
)

func (m *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/user/profile" { // на сколько правильная идея делать сравнение именно так
		var param ProfileParams
		param.Login = r.URL.Query().Get("profile")
		m.Profile(r.Context(), param)
	} else if r.URL.Path == "/user/create/" {
		age, _ := strconv.Atoi(r.URL.Query().Get("age"))
		param := CreateParams{
			Login:  r.URL.Query().Get("login"),
			Name:   r.URL.Query().Get("name"),
			Status: r.URL.Query().Get("status"),
			Age:    age,
		}
		m.Create(r.Context(), param)
	}
}
