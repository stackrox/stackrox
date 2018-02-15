package routes

import (
	"net/http"

	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/deny"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/NYTimes/gziphandler"
)

var (
	logger = logging.New("grpc/routes")
)

func defaultInterceptor(h http.Handler) http.Handler {
	return h
}

func authorizerHandler(h http.Handler, authorizer authz.Authorizer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := authorizer.Authorized(req.Context()); err != nil {
			writeHTTPStatus(w, err)
			return
		}
		h.ServeHTTP(w, req)
	})
}

// CustomRoute is a route that is directly accessed via HTTP
type CustomRoute struct {
	AuthInterceptor func(h http.Handler) http.Handler
	Authorizer      authz.Authorizer
	ServerHandler   http.Handler
	Compression     bool
}

// Handler is the http.Handler for the CustomRoute
func (c CustomRoute) Handler() http.Handler {
	if c.AuthInterceptor == nil {
		c.AuthInterceptor = defaultInterceptor
	}
	if c.Authorizer == nil {
		c.Authorizer = deny.Everyone()
	}
	h := c.AuthInterceptor(authorizerHandler(c.ServerHandler, c.Authorizer))
	if c.Compression {
		return gziphandler.GzipHandler(h)
	}
	return h
}
