package service

import (
	rolesDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	roleBindingsDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	as   Service
)

func initialize() {
	as = New(rolesDataStore.Singleton(), roleBindingsDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
