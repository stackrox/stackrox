package service

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the service interface.
type Service interface {
	grpc.APIService

	v1.FeatureFlagServiceServer
}

// New returns a new Service instance.
func New() Service {
	return &serviceImpl{}
}
