package private

import (
	"net/http"
)

type Route struct {
	Route         string
	ServerHandler http.Handler
	Compression   bool
}
