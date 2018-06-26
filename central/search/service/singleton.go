package service

import (
	"sync"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	policyDataStore "bitbucket.org/stack-rox/apollo/central/policy/datastore"
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
