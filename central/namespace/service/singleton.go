package service

import (
	"github.com/stackrox/rox/pkg/sync"

	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/namespace/datastore"
	networkPoliciesStore "github.com/stackrox/rox/central/networkpolicies/store"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
)

var (
	singleton Service
	once      sync.Once
)

// Singleton returns the singleton instance of the service.
func Singleton() Service {
	once.Do(func() {
		singleton = New(datastore.Singleton(), deploymentDataStore.Singleton(), secretDataStore.Singleton(), networkPoliciesStore.Singleton())
	})
	return singleton
}
