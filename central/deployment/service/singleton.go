package service

import (
	"github.com/stackrox/rox/central/deployment/datastore"
	lightspeedBroker "github.com/stackrox/rox/central/lightspeed/broker"
	lightspeedStore "github.com/stackrox/rox/central/lightspeed/store"
	processBaselineDataStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processBaselineResultsStore "github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), processIndicatorDataStore.Singleton(), processBaselineDataStore.Singleton(), processBaselineResultsStore.Singleton(), riskDataStore.Singleton(), manager.Singleton(),
		lightspeedStore.Singleton(), lightspeedBroker.Singleton(), connection.ManagerSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
