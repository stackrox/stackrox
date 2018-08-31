package service

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	deployTimeDetection "github.com/stackrox/rox/central/detection/deploytime"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/central/networkgraph"
	networkPolicyStore "github.com/stackrox/rox/central/networkpolicies/store"
	"github.com/stackrox/rox/central/risk"
	"github.com/stackrox/rox/central/secret/datastore"
	sensorEventDataStore "github.com/stackrox/rox/central/sensorevent/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(deployTimeDetection.SingletonDetector(), risk.GetScorer(), sensorEventDataStore.Singleton(), imageDataStore.Singleton(),
		deploymentDataStore.Singleton(), clusterDataStore.Singleton(), networkPolicyStore.Singleton(),
		namespaceStore.Singleton(), datastore.Singleton(), networkgraph.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
