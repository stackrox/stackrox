package service

import (
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	"github.com/stackrox/rox/central/reports/manager"
	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	collectionDS, _ := collectionDataStore.Singleton()
	svc = New(reportConfigDS.Singleton(), notifierDataStore.Singleton(), collectionDS, manager.Singleton())
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
