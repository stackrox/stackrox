package service

import (
	"sync"

	clusterDatastore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	"bitbucket.org/stack-rox/apollo/central/detection"
	"bitbucket.org/stack-rox/apollo/central/imageintegration"
	"bitbucket.org/stack-rox/apollo/central/imageintegration/datastore"
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
		detection.GetDetector())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
