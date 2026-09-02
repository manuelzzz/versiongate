package httpserver

import "net/http"

type echoRequest struct {
	Name string `json:"name"`
}

func testEchoHandler(w http.ResponseWriter, r *http.Request) {
	var body echoRequest
	if !DecodeJSON(w, r, &body) {
		return
	}
	if body.Name == "" {
		WriteError(w, CodeValidationError, "name is required")
		return
	}
	if body.Name == "missing" {
		WriteError(w, CodeNotFound, "no such name")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"name": body.Name})
}
