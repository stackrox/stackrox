package service

import (
	notifierDataStore "github.com/stackrox/stackrox/central/notifier/datastore"
	reportConfigDS "github.com/stackrox/stackrox/central/reportconfigurations/datastore"
	"github.com/stackrox/stackrox/central/reports/manager"
	accessScopeStore "github.com/stackrox/stackrox/central/role/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	svc = New(reportConfigDS.Singleton(), notifierDataStore.Singleton(), accessScopeStore.Singleton(), manager.Singleton())
}

// Singleton provides the instance of the service to register.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
