package service

import (
	"sync"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	deploymenteventStore "bitbucket.org/stack-rox/apollo/central/deploymentevent/store"
	detection "bitbucket.org/stack-rox/apollo/central/detection/singletons"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	risk "bitbucket.org/stack-rox/apollo/central/risk/singletons"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(detection.GetDetector(), risk.GetScorer(), deploymenteventStore.Singleton(), imageDataStore.Singleton(), deploymentDataStore.Singleton(), clusterDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
