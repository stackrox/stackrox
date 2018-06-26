package service

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/cluster/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
