package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			WriteError(w, CodeValidationError, "request body is required")
		} else {
			WriteError(w, CodeValidationError, "request body is not valid JSON")
		}
		return false
	}
	return true
}
