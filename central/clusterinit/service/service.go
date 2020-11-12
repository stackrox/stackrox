package service

import (
	"context"

	"github.com/stackrox/rox/central/clusterinit/backend"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the service for managing bootstrap tokens.
type Service interface {
	grpc.APIService
	v1.ClusterInitServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a new Service instance.
func New(backend backend.Backend) Service {
	return &serviceImpl{backend: backend}
}
