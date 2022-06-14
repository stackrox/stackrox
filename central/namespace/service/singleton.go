package service

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace/datastore"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	singleton Service
	once      sync.Once
)

// Singleton returns the singleton instance of the service.
func Singleton() Service {
	once.Do(func() {
		singleton = New(datastore.Singleton(), deploymentDataStore.Singleton(), secretDataStore.Singleton(), npDS.Singleton())
	})
	return singleton
}
