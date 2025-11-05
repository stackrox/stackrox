package retry

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ShouldRetryGrpcStatus(grpcStatus *status.Status) bool {
	code := grpcStatus.Code()
	switch code {
	case codes.Unavailable, codes.ResourceExhausted, codes.DeadlineExceeded:
		return true
	default:
		return false
	}
}
