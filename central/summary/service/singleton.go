package service

import (
	"sync"

	alertDataStore "bitbucket.org/stack-rox/apollo/central/alert/datastore"
	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(alertDataStore.Singleton(), clusterDataStore.Singleton(), deploymentDataStore.Singleton(), imageDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
