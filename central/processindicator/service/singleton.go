package service

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	whitelistDataStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(processIndicatorDataStore.Singleton(), deploymentDataStore.Singleton(), whitelistDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
