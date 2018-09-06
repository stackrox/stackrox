package service

import (
	"sync"

	"github.com/stackrox/rox/central/deployment/datastore"
	"github.com/stackrox/rox/central/enrichment"
	multiplierStore "github.com/stackrox/rox/central/multiplier/store"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), processIndicatorDataStore.Singleton(), multiplierStore.Singleton(), enrichment.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
