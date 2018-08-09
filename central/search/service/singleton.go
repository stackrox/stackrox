package service

import (
	"sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	policyDataStore "github.com/stackrox/rox/central/policy/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(alertDataStore.Singleton(), deploymentDataStore.Singleton(), imageDataStore.Singleton(), policyDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
