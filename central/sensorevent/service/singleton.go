package service

import (
	"sync"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/central/detection"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	namespaceStore "bitbucket.org/stack-rox/apollo/central/namespace/store"
	networkPolicyStore "bitbucket.org/stack-rox/apollo/central/networkpolicies/store"
	"bitbucket.org/stack-rox/apollo/central/risk"
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
