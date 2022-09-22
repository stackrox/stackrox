package httputil

import (
	"net/http"

	"github.com/stackrox/rox/pkg/env"
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

// NotImplementedOnManagedServices takes a handler func and returns a handler func that will error out if
// managed services env variable is set, otherwise it will return the given handler func
func NotImplementedOnManagedServices(fn http.Handler) http.Handler {
	if !env.ManagedCentral.BooleanSetting() {
		return fn
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		errMsg := "api is not supported in a managed central environment."
		log.Error(errMsg)
		http.Error(w, errMsg, http.StatusNotImplemented)
	})
}
