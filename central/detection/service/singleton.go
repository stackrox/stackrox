package service

import (
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	cveDataStore "github.com/stackrox/rox/central/cve/datastore"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/deploytime"
	"github.com/stackrox/rox/central/enrichment"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(clusterDatastore.Singleton(), enrichment.ImageEnricherSingleton(),
		imageDatastore.Singleton(),
		manager.Singleton(),
		cveDataStore.Singleton(),
		enrichment.Singleton(),
		buildTimeDetection.SingletonDetector(),
		deploytime.SingletonDetector(),
		deploytime.SingletonPolicySet())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
