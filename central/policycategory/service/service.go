package service

import (
	"context"

	"github.com/stackrox/rox/central/policycategory/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves policy categories data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.PolicyCategoryServiceServer
}

// New returns a new Service instance using the given DataStore.
func New(policyCategoriesDatastore datastore.DataStore) Service {
	return &serviceImpl{
		policyCategoriesDatastore: policyCategoriesDatastore,
	}
}
