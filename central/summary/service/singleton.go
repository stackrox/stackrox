package service

import (
	alertDataStore "github.com/stackrox/stackrox/central/alert/datastore"
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	imageDataStore "github.com/stackrox/stackrox/central/image/datastore"
	nodeDataStore "github.com/stackrox/stackrox/central/node/globaldatastore"
	secretDataStore "github.com/stackrox/stackrox/central/secret/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(alertDataStore.Singleton(), clusterDataStore.Singleton(),
		deploymentDataStore.Singleton(), imageDataStore.Singleton(),
		secretDataStore.Singleton(), nodeDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
