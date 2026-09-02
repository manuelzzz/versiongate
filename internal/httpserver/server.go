package httpserver

import "net/http"

func New() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /internal/_test-echo", testEchoHandler)
	return mux
}
