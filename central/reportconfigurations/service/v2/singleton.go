package v2

import (
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/rox/central/reportconfigurations/datastore"
	schedulerV2 "github.com/stackrox/rox/central/reports/scheduler/v2"
	collectionDataStore "github.com/stackrox/rox/central/resourcecollection/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	collectionDS, _ := collectionDataStore.Singleton()
	svc = New(reportConfigDS.Singleton(), notifierDataStore.Singleton(), collectionDS, schedulerV2.Singleton())
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
