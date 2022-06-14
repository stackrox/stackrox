package routes

import (
	"net/http"

	"github.com/stackrox/stackrox/pkg/grpc/errors"
)

func writeHTTPStatus(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	http.Error(w, err.Error(), errors.ErrToHTTPStatus(err))
}
