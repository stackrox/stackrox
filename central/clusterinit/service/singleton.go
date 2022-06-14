package service

import (
	clusterDataStore "github.com/stackrox/stackrox/central/cluster/datastore"
	"github.com/stackrox/stackrox/central/clusterinit/backend"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	svc Service
)

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(func() {
		svc = New(backend.Singleton(), clusterDataStore.Singleton())
	})
	return svc
}
