package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	graphConfigDataStore "github.com/stackrox/rox/central/networkflow/config/datastore"
	nfDS "github.com/stackrox/rox/central/networkflow/datastore"
	entityDataStore "github.com/stackrox/rox/central/networkflow/datastore/entities"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(nfDS.Singleton(),
		entityDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		clusterDataStore.Singleton(),
		graphConfigDataStore.Singleton(),
		connection.ManagerSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
