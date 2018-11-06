package service

import (
	"sync"

	alertDataStore "github.com/stackrox/rox/central/alert/datastore"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/node/store"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(alertDataStore.Singleton(), clusterDataStore.Singleton(),
		deploymentDataStore.Singleton(), imageDataStore.Singleton(),
		secretDataStore.Singleton(), store.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
