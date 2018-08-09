package service

import (
	"github.com/stackrox/rox/pkg/grpc/routes"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ReturnErrorCode is a helper function that formats an error with a status code if one is not already
// available.
func ReturnErrorCode(err error) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(routes.StatusError); ok {
		return status.Error(e.Status(), e.Error())
	}

	return status.Error(codes.Internal, err.Error())
}
