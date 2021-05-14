package service

import (
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	namespaceDS "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	svc = New(datastore.Singleton(), clusterDS.Singleton(), namespaceDS.Singleton())
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
