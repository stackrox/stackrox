package httputil

import "net/http"

// HTTPInterceptor is an interceptor function for HTTP handlers.
type HTTPInterceptor func(http.Handler) http.Handler

type interceptorChain []HTTPInterceptor

func (c interceptorChain) intercept(handler http.Handler) http.Handler {
	currHandler := handler
	for i := len(c) - 1; i >= 0; i-- {
		currHandler = c[i](currHandler)
	}
	return currHandler
}

// ChainInterceptors combines the given interceptors such that the first element in the list will be the first to
// process a request.
func ChainInterceptors(interceptors ...HTTPInterceptor) HTTPInterceptor {
	return interceptorChain(interceptors).intercept
}

// RoundTripInterceptor is an interceptor function for [RoundTripperFunc].
type RoundTripInterceptor func(req *http.Request, roundTrip RoundTripperFunc) (*http.Response, error)

// ChainRoundTripInterceptors combines the given interceptors such that the first element in the list will be the first to
// process a request.
func ChainRoundTripInterceptors(interceptors ...RoundTripInterceptor) RoundTripInterceptor {
	if len(interceptors) == 0 {
		return nil
	}
	return func(req *http.Request, roundTrip RoundTripperFunc) (*http.Response, error) {
		return interceptors[0](req, getRoundTrip(interceptors, 0, roundTrip))
	}
}

// getRoundTrip chains the given interceptors, recursively.
// This implementation is strongly based on https://github.com/grpc/grpc-go/blob/v1.60.1/server.go#L1187.
func getRoundTrip(interceptors []RoundTripInterceptor, curr int, finalRoundTrip RoundTripperFunc) RoundTripperFunc {
	if curr == len(interceptors) - 1 {
		return finalRoundTrip
	}
	return func(req *http.Request) (*http.Response, error) {
		return interceptors[curr+1](req, getRoundTrip(interceptors, curr+1, finalRoundTrip))
	}
}
