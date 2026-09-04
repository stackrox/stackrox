package service

import (
	aiIntegrationDS "github.com/stackrox/rox/central/aiintegration/datastore"
	"github.com/stackrox/rox/central/deployment/datastore"
	olsClient "github.com/stackrox/rox/central/lightspeed/client"
	processBaselineDataStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processBaselineResultsStore "github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	resolver := olsClient.NewIntegrationResolver(aiIntegrationDS.Singleton())
	as = New(datastore.Singleton(), processIndicatorDataStore.Singleton(), processBaselineDataStore.Singleton(), processBaselineResultsStore.Singleton(), riskDataStore.Singleton(), manager.Singleton(), olsClient.NewClient(resolver))
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
