package service

import (
	deploymentStore "github.com/stackrox/rox/central/deployment/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/datastore"
	roleDatastore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingDatastore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	saDatastore "github.com/stackrox/rox/central/serviceaccount/datastore"

	"github.com/stackrox/rox/pkg/sync"
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
