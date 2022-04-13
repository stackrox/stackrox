package service

import (
	"context"

	clusterDS "github.com/stackrox/stackrox/central/cluster/datastore"
	namespaceDS "github.com/stackrox/stackrox/central/namespace/datastore"
	"github.com/stackrox/stackrox/central/role/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service provides the interface to the gRPC service for roles.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	v1.RoleServiceServer
}

// New returns a new instance of the service. Please use the Singleton instead.
func New(roleDataStore datastore.DataStore, clusterDataStore clusterDS.DataStore, namespaceDataStore namespaceDS.DataStore) Service {
	return &serviceImpl{
		roleDataStore:      roleDataStore,
		clusterDataStore:   clusterDataStore,
		namespaceDataStore: namespaceDataStore,
	}
}
