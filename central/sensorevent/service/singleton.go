package service

import (
	"sync"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	detection "bitbucket.org/stack-rox/apollo/central/detection/singletons"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	namespaceStore "bitbucket.org/stack-rox/apollo/central/namespace/store"
	networkPolicyStore "bitbucket.org/stack-rox/apollo/central/networkpolicies/store"
	risk "bitbucket.org/stack-rox/apollo/central/risk/singletons"
	sensorEventDataStore "bitbucket.org/stack-rox/apollo/central/sensorevent/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(detection.GetDetector(), risk.GetScorer(), sensorEventDataStore.Singleton(), imageDataStore.Singleton(),
		deploymentDataStore.Singleton(), clusterDataStore.Singleton(), networkPolicyStore.Singleton(),
		namespaceStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
