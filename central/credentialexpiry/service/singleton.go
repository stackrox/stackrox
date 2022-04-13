package service

import (
	imageIntegrationStore "github.com/stackrox/stackrox/central/imageintegration/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	instance Service
)

func initialize() {
	instance = New(imageIntegrationStore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return instance
}
