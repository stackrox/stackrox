package service

import (
	"github.com/stackrox/rox/central/baseimage/datastore/repository"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	delegatedRegistryConfigDS "github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	"github.com/stackrox/rox/central/delegatedregistryconfig/delegator"
	"github.com/stackrox/rox/central/delegatedregistryconfig/scanwaiter"
	"github.com/stackrox/rox/central/delegatedregistryconfig/scanwaiterv2"
	"github.com/stackrox/rox/central/imageintegration"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	scanDelegator := delegator.New(
		delegatedRegistryConfigDS.Singleton(),
		connection.ManagerSingleton(),
		scanwaiter.Singleton(),
		scanwaiterv2.Singleton(),
		sachelper.NewClusterNamespaceSacHelper(clusterDataStore.Singleton(), namespaceDataStore.Singleton()),
	)

	as = New(repository.Singleton(), imageintegration.Set(), scanDelegator)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
