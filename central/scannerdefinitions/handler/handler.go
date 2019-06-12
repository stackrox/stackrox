package handler

import (
	"net/http"
)

// New returns a new instance of the service handler.
func New() http.Handler {
	return http.HandlerFunc(serveHTTP)
}
