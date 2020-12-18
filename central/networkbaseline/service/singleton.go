package service

import (
	"github.com/stackrox/rox/central/networkbaseline/datastore"
	"github.com/stackrox/rox/central/networkbaseline/manager"
	"github.com/stackrox/rox/pkg/sync"
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
