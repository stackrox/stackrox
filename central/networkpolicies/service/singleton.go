package service

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/networkgraph"
	"bitbucket.org/stack-rox/apollo/central/networkpolicies/store"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(store.Singleton(), networkgraph.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
