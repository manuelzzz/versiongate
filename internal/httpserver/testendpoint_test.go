package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestTestEcho exercises the shared error envelope and code/status
// mapping end-to-end through a full HTTP round trip, per issue #18's
// acceptance criteria.
func TestTestEcho(t *testing.T) {
	srv := httptest.NewServer(New())
	defer srv.Close()

	t.Run("valid request echoes name", func(t *testing.T) {
		resp := postJSON(t, srv.URL, echoRequest{Name: "alice"})
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", ct)
		}

		var body map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode response body: %v", err)
		}
		if body["name"] != "alice" {
			t.Fatalf("body[name] = %q, want alice", body["name"])
		}
	})

	t.Run("empty name is a validation error", func(t *testing.T) {
		resp := postJSON(t, srv.URL, echoRequest{Name: ""})
		defer resp.Body.Close()

		assertErrorResponse(t, resp, http.StatusBadRequest, CodeValidationError)
	})

	t.Run("missing name is a not-found error", func(t *testing.T) {
		resp := postJSON(t, srv.URL, echoRequest{Name: "missing"})
		defer resp.Body.Close()

		assertErrorResponse(t, resp, http.StatusNotFound, CodeNotFound)
	})

	t.Run("malformed body is a validation error", func(t *testing.T) {
		resp, err := http.Post(srv.URL+"/internal/_test-echo", "application/json", bytes.NewBufferString("{not json"))
		if err != nil {
			t.Fatalf("POST failed: %v", err)
		}
		defer resp.Body.Close()

		assertErrorResponse(t, resp, http.StatusBadRequest, CodeValidationError)
	})
}

func postJSON(t *testing.T, baseURL string, body echoRequest) *http.Response {
	t.Helper()

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("encode request body: %v", err)
	}

	resp, err := http.Post(baseURL+"/internal/_test-echo", "application/json", &buf)
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	return resp
}

func assertErrorResponse(t *testing.T, resp *http.Response, wantStatus int, wantCode ErrorCode) {
	t.Helper()

	if resp.StatusCode != wantStatus {
		t.Fatalf("status = %d, want %d", resp.StatusCode, wantStatus)
	}

	var body errorEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	if body.Error.Code != wantCode {
		t.Fatalf("body.Error.Code = %q, want %q", body.Error.Code, wantCode)
	}
}
