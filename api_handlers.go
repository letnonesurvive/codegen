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
func (s MyApi) handleprofile (w http.ResponseWriter, r *http.Request) { 

	var validator ApiValidator
	var profile ProfileParams
	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	query, _ := url.ParseQuery(string(bodyBytes))
	err := validator.Decode(&profile, query)
	if err != nil {
		WriteError(w, err)
		return
	}
	
}
func (s MyApi) handlecreate (w http.ResponseWriter, r *http.Request) { 
	if r.Method != http.MethodPost {
		WriteError(w, ApiError{HTTPStatus: http.StatusNotAcceptable, Err: fmt.Errorf("bad method")})
		return
	}
	var validator ApiValidator
	var create CreateParams
	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	query, _ := url.ParseQuery(string(bodyBytes))
	err := validator.Decode(&create, query)
	if err != nil {
		WriteError(w, err)
		return
	}
	
}
func (s OtherApi) handlecreate (w http.ResponseWriter, r *http.Request) { 
	if r.Method != http.MethodPost {
		WriteError(w, ApiError{HTTPStatus: http.StatusNotAcceptable, Err: fmt.Errorf("bad method")})
		return
	}
	var validator ApiValidator
	var create CreateParams
	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	query, _ := url.ParseQuery(string(bodyBytes))
	err := validator.Decode(&create, query)
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
		WriteError(w, ApiError{HTTPStatus: http.StatusNotFound, Err: fmt.Errorf("unknown method")})
	}
}
func (s *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/user/create":
		s.handlecreate(w, r) 
	default:
		WriteError(w, ApiError{HTTPStatus: http.StatusNotFound, Err: fmt.Errorf("unknown method")})
	}
}

