package service

import (
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

var (
	_ v1.LicenseServiceServer = (*service)(nil)
)

// New creates a new license service
func New() grpc.APIService {
	return newService()
}
