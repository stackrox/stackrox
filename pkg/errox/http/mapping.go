package http

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/pkg/errox/grpc"
)

// ErrToHTTPStatus maps known internal and gRPC errors to the appropriate
// HTTP status code.
func ErrToHTTPStatus(err error) int {
	return runtime.HTTPStatusFromCode(grpc.ErrToGRPCStatus(err).Code())
}
