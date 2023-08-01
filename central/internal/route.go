package internal

import (
	"net/http"
)

// Route is a route that is directly accessed via HTTP on cluster-internal server.
type Route struct {
	Route         string
	ServerHandler http.Handler
	Compression   bool
}
