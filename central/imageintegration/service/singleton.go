package service

import (
	clusterDatastore "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/enrichment"
	"github.com/stackrox/stackrox/central/imageintegration"
	"github.com/stackrox/stackrox/central/imageintegration/datastore"
	"github.com/stackrox/stackrox/central/reprocessor"
	"github.com/stackrox/stackrox/pkg/sync"
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
		reprocessor.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
