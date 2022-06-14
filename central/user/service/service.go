package service

import (
	"context"

	"github.com/stackrox/rox/central/user/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the gRPC service for users.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
	v1.UserServiceServer
}

// New returns a new instance of the service. Please use the Singleton instead.
func New(users datastore.DataStore) Service {
	return &serviceImpl{
		users: users,
	}
}
