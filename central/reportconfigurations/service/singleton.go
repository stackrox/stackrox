package service

import (
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reports/manager"
	collectionDS "github.com/stackrox/rox/central/resourcecollection/datastore"
	accessScopeStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	collectionDataStore, collectionQueryResolver := collectionDS.Singleton()
	svc = New(reportConfigDS.Singleton(), notifierDataStore.Singleton(), accessScopeStore.Singleton(), collectionDataStore, collectionQueryResolver, manager.Singleton())
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
