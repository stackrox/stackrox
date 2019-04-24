package service

import (
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	whitelistDataStore "github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	if features.ProcessWhitelist.Enabled() {
		as = New(processIndicatorDataStore.Singleton(), deploymentDataStore.Singleton(), whitelistDataStore.Singleton())
	} else { // Avoid calling any whitelist code if the feature has been turned off
		as = New(processIndicatorDataStore.Singleton(), deploymentDataStore.Singleton(), nil)
	}
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
