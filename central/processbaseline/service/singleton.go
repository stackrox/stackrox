package service

import (
	processBaselineDataStore "github.com/stackrox/stackrox/central/processbaseline/datastore"
	"github.com/stackrox/stackrox/central/reprocessor"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(processBaselineDataStore.Singleton(), reprocessor.Singleton(), connection.ManagerSingleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
