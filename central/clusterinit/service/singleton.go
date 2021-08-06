package service

import (
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusterinit/backend"
	"github.com/stackrox/rox/pkg/sync"
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
