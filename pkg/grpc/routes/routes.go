package routes

import (
	"fmt"
	"net/http"

	"github.com/NYTimes/gziphandler"
	"github.com/stackrox/stackrox/pkg/grpc/authz"
	"github.com/stackrox/stackrox/pkg/grpc/authz/deny"
	"github.com/stackrox/stackrox/pkg/httputil"
)

func authorizerHandler(h http.Handler, authorizer authz.Authorizer, postAuthInterceptor httputil.HTTPInterceptor, route string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := authorizer.Authorized(req.Context(), RPCNameForHTTP(route, req.Method)); err != nil {
			writeHTTPStatus(w, err)
			return
		}
		postAuthInterceptor(h).ServeHTTP(w, req)
	})
}

// CustomRoute is a route that is directly accessed via HTTP
type CustomRoute struct {
	Route         string
	Authorizer    authz.Authorizer
	ServerHandler http.Handler
	Compression   bool
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
func (c CustomRoute) Handler(postAuthInterceptor httputil.HTTPInterceptor) http.Handler {
	if c.Authorizer == nil {
		c.Authorizer = deny.Everyone()
	}
	h := authorizerHandler(c.ServerHandler, c.Authorizer, postAuthInterceptor, c.Route)
	if c.Compression {
		return gziphandler.GzipHandler(h)
	}
	return h
}
