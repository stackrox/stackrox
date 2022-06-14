package service

import (
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	buildTimeDetection "github.com/stackrox/stackrox/central/detection/buildtime"
	"github.com/stackrox/stackrox/central/detection/lifecycle"
	"github.com/stackrox/stackrox/central/enrichment"
	mitreDataStore "github.com/stackrox/stackrox/central/mitre/datastore"
	networkPolicyDS "github.com/stackrox/stackrox/central/networkpolicies/datastore"
	notifierDataStore "github.com/stackrox/stackrox/central/notifier/datastore"
	notifierProcessor "github.com/stackrox/stackrox/central/notifier/processor"
	"github.com/stackrox/stackrox/central/policy/datastore"
	"github.com/stackrox/stackrox/central/reprocessor"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(),
		clusterDataStore.Singleton(),
		deploymentDataStore.Singleton(),
		networkPolicyDS.Singleton(),
		notifierDataStore.Singleton(),
		mitreDataStore.Singleton(),
		reprocessor.Singleton(),
		buildTimeDetection.SingletonPolicySet(),
		lifecycle.SingletonManager(),
		notifierProcessor.Singleton(),
		enrichment.ImageMetadataCacheSingleton(),
		connection.ManagerSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
