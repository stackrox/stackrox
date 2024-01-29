package service

import (
	"context"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusterinit/backend"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
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
