package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	dataStore "github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(dataStore.Singleton(), clusterDataStore.Singleton(), connection.ManagerSingleton())

}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
