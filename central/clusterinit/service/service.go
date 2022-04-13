package service

import (
	"context"

	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/clusterinit/backend"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
	"github.com/stackrox/stackrox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the service for managing cluster init bundles.
type Service interface {
	grpc.APIService
	v1.ClusterInitServiceServer

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

// New returns a new Service instance.
func New(backend backend.Backend, clusterStore clusterDataStore.DataStore) Service {
	return &serviceImpl{
		backend:      backend,
		clusterStore: clusterStore,
	}
}
