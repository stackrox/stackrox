package service

import (
	"context"

	"github.com/stackrox/rox/central/serviceaccount/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves service account data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ServiceAccountServiceServer
}

// New returns a new Service instance using the given DB and index.
func New(serviceAccounts datastore.DataStore) Service {
	return &serviceImpl{
		storage: serviceAccounts,
	}
}
