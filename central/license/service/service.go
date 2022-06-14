package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

var (
	_ v1.LicenseServiceServer = (*service)(nil)
)

// New creates a new license service
func New() grpc.APIService {
	return newService()
}
