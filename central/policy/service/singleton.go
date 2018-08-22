package service

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	deployTimeDetection "github.com/stackrox/rox/central/detection/deploytime"
	runTimeDetectiomn "github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/enrichanddetect"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	notifierStore "github.com/stackrox/rox/central/notifier/store"
	"github.com/stackrox/rox/central/policy/datastore"
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
		buildTimeDetection.SingletonPolicySet(),
		deployTimeDetection.SingletonDetector(),
		runTimeDetectiomn.SingletonPolicySet(),
		notifierProcessor.Singleton(),
		enrichanddetect.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
