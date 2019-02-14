package service

import (
	"sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	"github.com/stackrox/rox/pkg/search/enumregistry"
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
		enumregistry.Singleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
