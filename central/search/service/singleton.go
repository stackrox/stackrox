package service

import (
	"github.com/stackrox/rox/pkg/sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/compliance/aggregation"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globalstore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(
		alertDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		imageDataStore.Singleton(),
		policyDataStore.Singleton(),
		secretDataStore.Singleton(),
		nodeDataStore.Singleton(),
		namespaceDataStore.Singleton(),
		aggregation.Singleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
