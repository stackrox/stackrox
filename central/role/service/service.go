package service

import (
	"context"

	"github.com/stackrox/rox/central/role/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the gRPC service for roles.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	v1.RoleServiceServer
}

// New returns a new instance of the service. Please use the Singleton instead.
func New(roleDataStore datastore.DataStore) Service {
	return &serviceImpl{
		roleDataStore: roleDataStore,
	}
}
