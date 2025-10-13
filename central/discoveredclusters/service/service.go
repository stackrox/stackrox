package service

import (
	"context"

	"github.com/stackrox/rox/central/discoveredclusters/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the gRPC service for users.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.DiscoveredClustersServiceServer
}

func newService(datastore datastore.DataStore) Service {
	return &serviceImpl{
		ds: datastore,
	}
}
