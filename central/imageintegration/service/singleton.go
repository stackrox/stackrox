package service

import (
	"sync"

	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/enrichanddetect"
	"github.com/stackrox/rox/central/imageintegration"
	"github.com/stackrox/rox/central/imageintegration/datastore"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	as = New(imageintegration.Set().RegistryFactory(),
		imageintegration.Set().ScannerFactory(),
		imageintegration.ToNotify(),
		datastore.Singleton(),
		clusterDatastore.Singleton(),
		enrichanddetect.GetLoop())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
