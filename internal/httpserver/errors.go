package httpserver

import "net/http"

type ErrorCode string

const (
	CodeValidationError ErrorCode = "validation_error"
	CodeUnauthorized    ErrorCode = "unauthorized"
	CodeNotFound        ErrorCode = "not_found"
	CodeConflict        ErrorCode = "conflict"
	CodeInternalError   ErrorCode = "internal_error"
)

var statusByCode = map[ErrorCode]int{
	CodeValidationError: http.StatusBadRequest,
	CodeUnauthorized:    http.StatusUnauthorized,
	CodeNotFound:        http.StatusNotFound,
	CodeConflict:        http.StatusConflict,
	CodeInternalError:   http.StatusInternalServerError,
}

type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

type errorEnvelope struct {
	Error APIError `json:"error"`
}

func WriteError(w http.ResponseWriter, code ErrorCode, message string) {
	status, ok := statusByCode[code]
	if !ok {
		status = http.StatusInternalServerError
	}
	writeJSON(w, status, errorEnvelope{Error: APIError{Code: code, Message: message}})
}
