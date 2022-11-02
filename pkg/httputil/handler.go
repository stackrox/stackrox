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

// HandlerNotImplementedIfSettingDisabled takes a handler func and returns a handler func that will error out if
// specified boolean is disabled, otherwise it will return the given handler func
func HandlerNotImplementedIfSettingDisabled(enabled bool, fn http.Handler) http.Handler {
	if enabled {
		return fn
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		errMsg := "api is disabled due to central setting"
		log.Error(errMsg)
		http.Error(w, errMsg, http.StatusNotImplemented)
	})
}
