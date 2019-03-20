package service

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/stackrox/rox/central/deployment/datastore"
	multiplierStore "github.com/stackrox/rox/central/multiplier/store"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/risk/manager"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), processIndicatorDataStore.Singleton(), multiplierStore.Singleton(), manager.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
