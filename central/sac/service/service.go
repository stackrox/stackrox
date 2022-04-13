package service

import (
	"context"

	"github.com/stackrox/stackrox/central/sac/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service is the service which exposed the ability to create, edit, and remove auth plugin configurations.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ScopedAccessControlServiceServer
}

// New returns a new instance of a Service.
func New(ds datastore.DataStore) Service {
	return &serviceImpl{
		ds: ds,
	}
}
