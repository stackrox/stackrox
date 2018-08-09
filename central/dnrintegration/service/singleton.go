package service

import (
	"sync"

	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/dnrintegration/datastore"
	"github.com/stackrox/rox/central/enrichment"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), clusterDataStore.Singleton(), enrichment.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
