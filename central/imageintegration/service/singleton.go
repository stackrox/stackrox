package service

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/imageintegration"
	"github.com/stackrox/rox/central/imageintegration/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(imageintegration.Set().RegistryFactory(),
		imageintegration.Set().ScannerFactory(),
		enrichment.ManagerSingleton(),
		enrichment.NodeEnricherSingleton(),
		datastore.Singleton(),
		clusterDatastore.Singleton(),
		reprocessor.Singleton(),
		connection.ManagerSingleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
