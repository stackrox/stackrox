package service

import (
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	baselineDataStore "github.com/stackrox/stackrox/central/processbaseline/datastore"
	processIndicatorDataStore "github.com/stackrox/stackrox/central/processindicator/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(processIndicatorDataStore.Singleton(), deploymentDataStore.Singleton(), baselineDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
