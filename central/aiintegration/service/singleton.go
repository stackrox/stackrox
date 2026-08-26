package service

import (
	"github.com/stackrox/rox/central/aiintegration/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	serviceInstance     Service
	serviceInstanceInit sync.Once
)

// Singleton returns the singleton instance of the AI integration service.
func Singleton() Service {
	serviceInstanceInit.Do(func() {
		serviceInstance = New(datastore.Singleton())
	})
	return serviceInstance
}
