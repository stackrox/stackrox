package httputil

import (
	"net/http"
	"slices"
)

// HTTPInterceptor is an interceptor function for HTTP handlers.
type HTTPInterceptor func(http.Handler) http.Handler

type interceptorChain []HTTPInterceptor

func (c interceptorChain) intercept(handler http.Handler) http.Handler {
	currHandler := handler
	for _, h := range slices.Backward(c) {
		currHandler = h(currHandler)
	}
	return currHandler
}

// ChainInterceptors combines the given interceptors such that the first element in the list will be the first to
// process a request.
func ChainInterceptors(interceptors ...HTTPInterceptor) HTTPInterceptor {
	return interceptorChain(interceptors).intercept
}
