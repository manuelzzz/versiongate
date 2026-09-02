package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		code       ErrorCode
		message    string
		wantStatus int
	}{
		{name: "validation error", code: CodeValidationError, message: "bad input", wantStatus: http.StatusBadRequest},
		{name: "unauthorized", code: CodeUnauthorized, message: "no token", wantStatus: http.StatusUnauthorized},
		{name: "not found", code: CodeNotFound, message: "no such thing", wantStatus: http.StatusNotFound},
		{name: "conflict", code: CodeConflict, message: "already exists", wantStatus: http.StatusConflict},
		{name: "internal error", code: CodeInternalError, message: "something broke", wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			WriteError(rec, tt.code, tt.message)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}

			var body errorEnvelope
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode response body: %v", err)
			}
			if body.Error.Code != tt.code {
				t.Fatalf("body.Error.Code = %q, want %q", body.Error.Code, tt.code)
			}
			if body.Error.Message != tt.message {
				t.Fatalf("body.Error.Message = %q, want %q", body.Error.Message, tt.message)
			}
		})
	}
}
