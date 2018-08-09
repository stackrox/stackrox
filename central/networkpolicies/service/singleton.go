package service

import (
	"sync"

	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/networkgraph"
	"github.com/stackrox/rox/central/networkpolicies/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(store.Singleton(), networkgraph.Singleton(), datastore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
