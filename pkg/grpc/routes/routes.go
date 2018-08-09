package routes

import (
	"fmt"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/deny"
)

func defaultInterceptor(h http.Handler) http.Handler {
	return h
}

func authorizerHandler(h http.Handler, authorizer authz.Authorizer, route string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := authorizer.Authorized(req.Context(), RPCNameForHTTP(route, req.Method)); err != nil {
			writeHTTPStatus(w, err)
			return
		}
		h.ServeHTTP(w, req)
	})
}

// CustomRoute is a route that is directly accessed via HTTP
type CustomRoute struct {
	Route           string
	AuthInterceptor func(h http.Handler) http.Handler
	Authorizer      authz.Authorizer
	ServerHandler   http.Handler
	Compression     bool
}

// RPCNameForHTTP returns the RPCName to be used for this HTTP route.
// HTTP routes don't really have an RPC name, but we use this
// as a (hacky) equivalent so as to provider a clean abstraction
// to downstream methods (like authorizers).
func RPCNameForHTTP(route, method string) string {
	if method == "" {
		return route
	}
	return fmt.Sprintf("%s %s", method, route)
}

// Handler is the http.Handler for the CustomRoute
func (c CustomRoute) Handler() http.Handler {
	if c.AuthInterceptor == nil {
		c.AuthInterceptor = defaultInterceptor
	}
	if c.Authorizer == nil {
		c.Authorizer = deny.Everyone()
	}
	h := c.AuthInterceptor(authorizerHandler(c.ServerHandler, c.Authorizer, c.Route))
	if c.Compression {
		return gziphandler.GzipHandler(h)
	}
	return h
}
