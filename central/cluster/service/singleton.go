package service

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	"bitbucket.org/stack-rox/apollo/central/enrichment/singletons"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), singletons.GetEnricher())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
