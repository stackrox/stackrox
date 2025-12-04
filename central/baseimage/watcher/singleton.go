package watcher

import (
	"github.com/stackrox/rox/central/baseimage/datastore"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	delegatedRegistryConfigDS "github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	"github.com/stackrox/rox/central/delegatedregistryconfig/delegator"
	scanwaiterv2 "github.com/stackrox/rox/central/delegatedregistryconfig/scanWaiterV2"
	"github.com/stackrox/rox/central/delegatedregistryconfig/scanwaiter"
	"github.com/stackrox/rox/central/imageintegration"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	watcher Watcher
)

// Singleton returns the global base image watcher instance.
func Singleton() Watcher {
	once.Do(func() {
		scanDelegator := delegator.New(
			delegatedRegistryConfigDS.Singleton(),
			connection.ManagerSingleton(),
			scanwaiter.Singleton(),
			scanwaiterv2.Singleton(),
			sachelper.NewClusterNamespaceSacHelper(clusterDataStore.Singleton(), namespaceDataStore.Singleton()),
		)

		watcher = New(
			datastore.Singleton(),
			imageintegration.Set().RegistrySet(),
			scanDelegator,
		)
	})
	return watcher
}
