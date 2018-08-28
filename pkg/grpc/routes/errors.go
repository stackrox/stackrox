package routes

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/status"
)

func writeHTTPStatus(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	s := status.Convert(err)
	http.Error(w, s.Message(), runtime.HTTPStatusFromCode(s.Code()))
}
