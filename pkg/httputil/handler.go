package httputil

import (
	"net/http"
)

// WrapHandlerFunc wraps a function returning an error into an HTTP handler func that returns a 200 OK with empty
// contents upon success, and sends an error formatted according to `WriteError` to the client otherwise.
func WrapHandlerFunc(handlerFn func(req *http.Request) error) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err := handlerFn(req)
		if err != nil {
			WriteError(w, err)
			return
		}
	})
}

// NotImplementedHandler returns an HTTP Handler func that returns 501 Not Implemented with a custom error message.
func NotImplementedHandler(errMsg string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Error(errMsg)
		http.Error(w, errMsg, http.StatusNotImplemented)
	})
}
