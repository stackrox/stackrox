package routes

import (
	"net/http"

	"github.com/stackrox/rox/pkg/grpc/errors/grpccode"
)

func writeHTTPStatus(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	http.Error(w, err.Error(), grpccode.ErrToHTTPStatus(err))
}
