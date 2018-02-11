package service

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StatusError allows errors to be emitted with the proper status code.
type StatusError interface {
	error
	Status() codes.Code
}

func returnErrorCode(err error) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(StatusError); ok {
		return status.Error(e.Status(), e.Error())
	}

	return status.Error(codes.Internal, err.Error())
}
