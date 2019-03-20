package service

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/risk/manager"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), manager.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
