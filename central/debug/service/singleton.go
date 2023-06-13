package service

import (
	authProviderRegistry "github.com/stackrox/rox/central/authprovider/registry"
	"github.com/stackrox/rox/central/cluster/datastore"
	configDS "github.com/stackrox/rox/central/config/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	logimbueStore "github.com/stackrox/rox/central/logimbue/store"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/central/telemetry/gatherers"
	"github.com/stackrox/rox/central/trace"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	debugService Service
)

func initialize() {
	debugService = New(datastore.Singleton(),
		connection.ManagerSingleton(),
		gatherers.Singleton(),
		logimbueStore.Singleton(),
		trace.AuthzTraceSinkSingleton(),
		authProviderRegistry.Singleton(),
		groupDataStore.Singleton(),
		roleDataStore.Singleton(),
		configDS.Singleton(),
		notifierDS.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return debugService
}
