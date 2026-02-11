package service

import (
	clusterCVEDatastore "github.com/stackrox/rox/central/cve/cluster/datastore"
	imageCVEDatastore "github.com/stackrox/rox/central/cve/image/v2/datastore"
	nodeCVEDatastore "github.com/stackrox/rox/central/cve/node/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(imageCVEDatastore.Singleton(), nodeCVEDatastore.Singleton(), clusterCVEDatastore.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
