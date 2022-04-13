package service

import (
	deploymentStore "github.com/stackrox/stackrox/central/deployment/datastore"
	namespaceStore "github.com/stackrox/stackrox/central/namespace/datastore"
	roleDatastore "github.com/stackrox/stackrox/central/rbac/k8srole/datastore"
	bindingDatastore "github.com/stackrox/stackrox/central/rbac/k8srolebinding/datastore"
	saDatastore "github.com/stackrox/stackrox/central/serviceaccount/datastore"

	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once
	as   Service
)

func initialize() {
	as = New(saDatastore.Singleton(), bindingDatastore.Singleton(), roleDatastore.Singleton(), deploymentStore.Singleton(), namespaceStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
