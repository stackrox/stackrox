package service

import (
	"github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/probesources"
	"github.com/stackrox/stackrox/central/risk/manager"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(datastore.Singleton(), manager.Singleton(), probesources.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
