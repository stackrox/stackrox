package service

import (
	"github.com/stackrox/stackrox/central/networkbaseline/datastore"
	"github.com/stackrox/stackrox/central/networkbaseline/manager"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	service Service
)

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(func() {
		service = New(datastore.Singleton(), manager.Singleton())
	})
	return service
}
