package service

import (
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	"github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	as Service
)

func initialize() {
	ds, qr := datastore.Singleton()
	as = New(ds, qr, deploymentDS.Singleton(), reportConfigDS.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
