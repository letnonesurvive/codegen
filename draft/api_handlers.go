package draft

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type ProfileResponse struct {
	Error string `json:"error"`
	User  *User  `json:"response,omitempty"`
}

type CreateResponse struct {
	Error string   `json:"error"`
	User  *NewUser `json:"response,omitempty"`
}

func WriteError(w http.ResponseWriter, err error) {
	var response ProfileResponse
	if apiError, ok := err.(ApiError); ok {
		w.WriteHeader(apiError.HTTPStatus)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}

	response.Error = err.Error()
	data, _ := json.Marshal(response)
	w.Write(data)
}

func (m *MyApi) handleProfile(w http.ResponseWriter, r *http.Request) {
	var params ProfileParams
	var validator ApiValidator
	if r.Method == http.MethodPost {
		bodyBytes, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		query, _ := url.ParseQuery(string(bodyBytes))
		err := validator.Decode(&params, query)
		if err != nil {
			WriteError(w, err)
			return
		}
	} else if r.Method == http.MethodGet {
		err := validator.Decode(&params, r.URL.Query())
		if err != nil {
			WriteError(w, err)
			return
		}
	}

	user, err := m.Profile(r.Context(), params)
	if err != nil {
		WriteError(w, err)
		return
	}

	var response ProfileResponse
	response.User = user
	data, err := json.Marshal(response)
	if err != nil {
		WriteError(w, ApiError{HTTPStatus: http.StatusInternalServerError, Err: err})
		return
	}

	w.Write(data)
}

func (m *MyApi) handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, ApiError{HTTPStatus: http.StatusNotAcceptable, Err: fmt.Errorf("bad method")})
		return
	}
	auth, ok := r.Header["X-Auth"]
	if !ok || auth[0] != "100500" {
		WriteError(w, ApiError{HTTPStatus: http.StatusForbidden, Err: fmt.Errorf("unauthorized")})
		return
	}

	var params CreateParams

	bodyBytes, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	query, _ := url.ParseQuery(string(bodyBytes))
	var validator ApiValidator
	err := validator.Decode(&params, query)

	if err != nil {
		WriteError(w, err)
		return
	}

	user, err := m.Create(r.Context(), params)
	if err != nil {
		WriteError(w, err)
		return
	}

	var response CreateResponse
	response.User = user
	data, _ := json.Marshal(response)
	w.Write(data)
}

func (m *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.URL.Path {
	case "/user/profile":
		m.handleProfile(w, r)
	case "/user/create":
		m.handleCreate(w, r)
	default:
		WriteError(w, ApiError{HTTPStatus: http.StatusNotFound, Err: fmt.Errorf("unknown method")})
	}
}
