package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func WriteError(w http.ResponseWriter, err error) {
	var response ErrorResponse
	if apiError, ok := err.(ApiError); ok {
		w.WriteHeader(apiError.HTTPStatus)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
	response.Error = err.Error()
	data, _ := json.Marshal(response)
	w.Write(data)
}
func (s MyApi) handleprofile(w http.ResponseWriter, r *http.Request) {

	var params ProfileParams
	var validator ApiValidator
	var query url.Values
	if r.Method == http.MethodPost {
		bodyBytes, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		query, _ = url.ParseQuery(string(bodyBytes))
	} else if r.Method == http.MethodGet {
		query = r.URL.Query()
	}
	err := validator.Decode(&params, query)
	if err != nil {
		WriteError(w, err)
		return
	}

}
func (s MyApi) handlecreate(w http.ResponseWriter, r *http.Request) {

	var params CreateParams
	var validator ApiValidator
	var query url.Values
	if r.Method != http.MethodPost {
		WriteError(w, ApiError{HTTPStatus: 406, Err: fmt.Errorf("bad method")})
		return
	}
	auth, ok := r.Header["X-Auth"]
	if !ok || auth[0] != "100500" {
		WriteError(w, ApiError{HTTPStatus: 403, Err: fmt.Errorf("unauthorized")})
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	query, _ = url.ParseQuery(string(bodyBytes))
	err := validator.Decode(&params, query)
	if err != nil {
		WriteError(w, err)
		return
	}

}
func (s OtherApi) handlecreate(w http.ResponseWriter, r *http.Request) {

	var params CreateParams
	var validator ApiValidator
	var query url.Values
	if r.Method != http.MethodPost {
		WriteError(w, ApiError{HTTPStatus: 406, Err: fmt.Errorf("bad method")})
		return
	}
	auth, ok := r.Header["X-Auth"]
	if !ok || auth[0] != "100500" {
		WriteError(w, ApiError{HTTPStatus: 403, Err: fmt.Errorf("unauthorized")})
		return
	}

	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	query, _ = url.ParseQuery(string(bodyBytes))
	err := validator.Decode(&params, query)
	if err != nil {
		WriteError(w, err)
		return
	}

}
func (s *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/user/profile":
		s.handleprofile(w, r)
	case "/user/create":
		s.handlecreate(w, r)
	default:
		WriteError(w, ApiError{HTTPStatus: 404, Err: fmt.Errorf("unknown method")})
	}
}
func (s *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/user/create":
		s.handlecreate(w, r)
	default:
		WriteError(w, ApiError{HTTPStatus: 404, Err: fmt.Errorf("unknown method")})
	}
}
