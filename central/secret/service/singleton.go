package service

import (
	deploymentDataStore "github.com/stackrox/stackrox/central/deployment/datastore"
	"github.com/stackrox/stackrox/central/secret/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once
	as   Service
)

func initialize() {
	as = New(datastore.Singleton(), deploymentDataStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
