package service

import (
	"bitbucket.org/stack-rox/apollo/central/db"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func returnErrorCode(err error) error {
	if err == nil {
		return nil
	}

	switch err.(type) {
	case db.ErrNotFound:
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
