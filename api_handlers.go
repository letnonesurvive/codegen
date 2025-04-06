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

type MyApiProfileResponse struct {
	Error string `json:"error"`
	User  *User  `json:"response,omitempty"`
}

func (s MyApi) handleProfile(w http.ResponseWriter, r *http.Request) {

	var params ProfileParams
	var response MyApiProfileResponse
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
	user, err := s.Profile(r.Context(), params)
	if err != nil {
		WriteError(w, err)
		return
	}
	response.User = user
	data, err := json.Marshal(response)
	if err != nil {
		WriteError(w, ApiError{HTTPStatus: 500, Err: fmt.Errorf("err")})
		return
	}
	w.Write(data)

}

type MyApiCreateResponse struct {
	Error string   `json:"error"`
	User  *NewUser `json:"response,omitempty"`
}

func (s MyApi) handleCreate(w http.ResponseWriter, r *http.Request) {

	var params CreateParams
	var response MyApiCreateResponse
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
	user, err := s.Create(r.Context(), params)
	if err != nil {
		WriteError(w, err)
		return
	}
	response.User = user
	data, err := json.Marshal(response)
	if err != nil {
		WriteError(w, ApiError{HTTPStatus: 500, Err: fmt.Errorf("err")})
		return
	}
	w.Write(data)

}

type OtherApiCreateResponse struct {
	Error string     `json:"error"`
	User  *OtherUser `json:"response,omitempty"`
}

func (s OtherApi) handleCreate(w http.ResponseWriter, r *http.Request) {

	var params OtherCreateParams
	var response OtherApiCreateResponse
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
	user, err := s.Create(r.Context(), params)
	if err != nil {
		WriteError(w, err)
		return
	}
	response.User = user
	data, err := json.Marshal(response)
	if err != nil {
		WriteError(w, ApiError{HTTPStatus: 500, Err: fmt.Errorf("err")})
		return
	}
	w.Write(data)

}
func (s *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/user/profile":
		s.handleProfile(w, r)
	case "/user/create":
		s.handleCreate(w, r)
	default:
		WriteError(w, ApiError{HTTPStatus: 404, Err: fmt.Errorf("unknown method")})
	}
}
func (s *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/user/create":
		s.handleCreate(w, r)
	default:
		WriteError(w, ApiError{HTTPStatus: 404, Err: fmt.Errorf("unknown method")})
	}
}
