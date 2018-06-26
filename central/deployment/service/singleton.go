package service

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	enrichment "bitbucket.org/stack-rox/apollo/central/enrichment/singletons"
	multiplierStore "bitbucket.org/stack-rox/apollo/central/multiplier/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), multiplierStore.Singleton(), enrichment.GetEnricher())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
