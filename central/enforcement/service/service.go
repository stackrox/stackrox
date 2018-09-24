package service

import (
	"context"

	"github.com/stackrox/rox/central/sensorevent/service/streamer"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that applies enforcement.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.EnforcementServiceServer
}

// New returns a new Service instance using the given DB and index.
func New(manager streamer.Manager) Service {
	return &serviceImpl{
		manager: manager,
	}
}
