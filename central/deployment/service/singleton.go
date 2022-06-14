package service

import (
	"github.com/stackrox/stackrox/central/deployment/datastore"
	processBaselineDataStore "github.com/stackrox/stackrox/central/processbaseline/datastore"
	processBaselineResultsStore "github.com/stackrox/stackrox/central/processbaselineresults/datastore"
	processIndicatorDataStore "github.com/stackrox/stackrox/central/processindicator/datastore"
	riskDataStore "github.com/stackrox/stackrox/central/risk/datastore"
	"github.com/stackrox/stackrox/central/risk/manager"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), processIndicatorDataStore.Singleton(), processBaselineDataStore.Singleton(), processBaselineResultsStore.Singleton(), riskDataStore.Singleton(), manager.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
