package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/central/enrichment"
	networkPolicyDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/sensor/service/connection"
	mitreDataStore "github.com/stackrox/rox/pkg/mitre/datastore"
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
