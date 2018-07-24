package service

import (
	"sync"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	"bitbucket.org/stack-rox/apollo/central/detection"
	notifierStore "bitbucket.org/stack-rox/apollo/central/notifier/store"
	"bitbucket.org/stack-rox/apollo/central/policy/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), clusterDataStore.Singleton(), deploymentDataStore.Singleton(), notifierStore.Singleton(), detection.GetDetector())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
