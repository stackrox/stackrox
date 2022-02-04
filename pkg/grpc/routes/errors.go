package routes

import (
	"net/http"

	errox_http "github.com/stackrox/rox/pkg/errox/http"
)

func writeHTTPStatus(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	http.Error(w, err.Error(), errox_http.ErrToHTTPStatus(err))
}
