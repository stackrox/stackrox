package service

import (
	"github.com/stackrox/rox/central/cloudsources/datastore"
	"github.com/stackrox/rox/central/cloudsources/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	svc Service
)

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(func() {
		svc = newService(datastore.Singleton(), manager.Singleton())
	})
	return svc
}
