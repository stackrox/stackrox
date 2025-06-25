package service

import (
	authProviderRegistry "github.com/stackrox/rox/central/authprovider/registry"
	"github.com/stackrox/rox/central/group/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(func() {
		svc = New(datastore.Singleton(), authProviderRegistry.Singleton())
	})
	return svc
}
