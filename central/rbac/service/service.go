package service

import (
	"context"

	rolesDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingsDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
)

// Service provides the interface to the microservice that serves secret data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.RbacServiceServer
}

// New returns a new Service instance using the given DB and index.
func New(roles rolesDataStore.DataStore, bindings roleBindingsDataStore.DataStore) Service {
	return &serviceImpl{
		roles:    roles,
		bindings: bindings,
	}
}
