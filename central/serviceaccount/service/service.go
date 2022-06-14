package service

import (
	"context"

	deploymentStore "github.com/stackrox/stackrox/central/deployment/datastore"
	namespaceStore "github.com/stackrox/stackrox/central/namespace/datastore"
	roleDatastore "github.com/stackrox/stackrox/central/rbac/k8srole/datastore"
	bindingDatastore "github.com/stackrox/stackrox/central/rbac/k8srolebinding/datastore"
	saDatastore "github.com/stackrox/stackrox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/grpc"
)

// Service provides the interface to the microservice that serves service account data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	v1.ServiceAccountServiceServer
}

// New returns a new Service instance using the given DB and index.
func New(serviceAccounts saDatastore.DataStore, rolebindings bindingDatastore.DataStore, roles roleDatastore.DataStore, deployments deploymentStore.DataStore, namespaces namespaceStore.DataStore) Service {
	return &serviceImpl{
		serviceAccounts: serviceAccounts,
		bindings:        rolebindings,
		roles:           roles,
		deployments:     deployments,
		namespaces:      namespaces,
	}
}
