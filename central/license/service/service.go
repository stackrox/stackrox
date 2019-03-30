package service

import (
	"github.com/stackrox/rox/pkg/grpc"
)

// New creates a new license service
func New(lockdownMode bool) grpc.APIService {
	return newService(lockdownMode)
}
