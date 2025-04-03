package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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
func (s MyApi) handleuserprofile(w http.ResponseWriter, r *http.Request) {

}
func (s MyApi) handleusercreate(w http.ResponseWriter, r *http.Request) {

}
func (s OtherApi) handleusercreate(w http.ResponseWriter, r *http.Request) {

}
func (s *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/user/profile":
		s.handleuserprofile(w, r)
	case "/user/create":
		s.handleusercreate(w, r)
	default:
		WriteError(w, ApiError{HTTPStatus: http.StatusNotFound, Err: fmt.Errorf("unknown method")})
	}
}
func (s *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/user/create":
		s.handleusercreate(w, r)
	default:
		WriteError(w, ApiError{HTTPStatus: http.StatusNotFound, Err: fmt.Errorf("unknown method")})
	}
}
