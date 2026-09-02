package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	t.Run("valid body decodes and ignores unknown fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"alice","extra":"ignored"}`))
		rec := httptest.NewRecorder()

		var dst echoRequest
		ok := DecodeJSON(rec, req, &dst)

		if !ok {
			t.Fatalf("DecodeJSON returned false, want true")
		}
		if dst.Name != "alice" {
			t.Fatalf("dst.Name = %q, want alice", dst.Name)
		}
	})

	t.Run("empty body is a validation error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		rec := httptest.NewRecorder()

		var dst echoRequest
		ok := DecodeJSON(rec, req, &dst)

		if ok {
			t.Fatalf("DecodeJSON returned true, want false")
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("malformed JSON is a validation error", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{not json"))
		rec := httptest.NewRecorder()

		var dst echoRequest
		ok := DecodeJSON(rec, req, &dst)

		if ok {
			t.Fatalf("DecodeJSON returned true, want false")
		}
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}
