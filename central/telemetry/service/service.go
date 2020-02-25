package service

import (
	"context"

	"github.com/stackrox/rox/central/telemetry/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service is the interface to the gRPC service for managing telemetry
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.TelemetryServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(mgr manager.Manager) Service {
	return &serviceImpl{
		manager: mgr,
	}
}
