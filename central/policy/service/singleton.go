package service

import (
	"github.com/stackrox/rox/pkg/sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/enrichanddetect"
	"github.com/stackrox/rox/central/enrichment"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	"github.com/stackrox/rox/central/policy/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(),
		clusterDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		notifierStore.Singleton(),
		processIndicatorDataStore.Singleton(),
		buildTimeDetection.SingletonPolicySet(),
		lifecycle.SingletonManager(),
		notifierProcessor.Singleton(),
		enrichanddetect.Singleton(),
		enrichment.ImageMetadataCacheSingleton(),
		enrichment.ImageScanCacheSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
