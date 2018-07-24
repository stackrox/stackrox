package routes

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
)

// StatusError allows errors to be emitted with the proper status code.
type StatusError interface {
	error
	Status() codes.Code
}

func writeHTTPStatus(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	if e, ok := err.(StatusError); ok {
		http.Error(w, e.Error(), runtime.HTTPStatusFromCode(e.Status()))
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
