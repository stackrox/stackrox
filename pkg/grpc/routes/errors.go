package routes

import "net/http"

// HTTPStatus allows errors to be emitted with the proper HTTP status code.
type HTTPStatus interface {
	error
	HTTPStatus() int
}

func writeHTTPStatus(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	if e, ok := err.(HTTPStatus); ok {
		http.Error(w, e.Error(), e.HTTPStatus())
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
