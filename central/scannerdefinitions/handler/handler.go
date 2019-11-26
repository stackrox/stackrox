package handler

import (
	"net/http"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// New returns a new instance of the service handler.
func New() http.Handler {
	return http.HandlerFunc(serveHTTP)
}
