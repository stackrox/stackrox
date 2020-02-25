package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/enrichment"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(),
		clusterDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		notifierDataStore.Singleton(),
		reprocessor.Singleton(),
		buildTimeDetection.SingletonPolicySet(),
		searchbasedpolicies.DeploymentBuilderSingleton(),
		searchbasedpolicies.ImageBuilderSingleton(),
		lifecycle.SingletonManager(),
		notifierProcessor.Singleton(),
		enrichment.ImageMetadataCacheSingleton(),
		enrichment.ImageScanCacheSingleton(),
		connection.ManagerSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
