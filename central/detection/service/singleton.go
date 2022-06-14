package service

import (
	clusterDatastore "github.com/stackrox/stackrox/central/cluster/datastore"
	buildTimeDetection "github.com/stackrox/stackrox/central/detection/buildtime"
	"github.com/stackrox/stackrox/central/detection/deploytime"
	"github.com/stackrox/stackrox/central/enrichment"
	imageDatastore "github.com/stackrox/stackrox/central/image/datastore"
	"github.com/stackrox/stackrox/central/notifier/processor"
	"github.com/stackrox/stackrox/central/risk/manager"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(clusterDatastore.Singleton(), enrichment.ImageEnricherSingleton(),
		imageDatastore.Singleton(),
		manager.Singleton(),
		enrichment.Singleton(),
		buildTimeDetection.SingletonDetector(),
		processor.Singleton(),
		deploytime.SingletonDetector(),
		deploytime.SingletonPolicySet())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
