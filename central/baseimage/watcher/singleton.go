package watcher

import (
	baseImageDS "github.com/stackrox/rox/central/baseimage/datastore"
	repoDS "github.com/stackrox/rox/central/baseimage/datastore/repository"
	tagDS "github.com/stackrox/rox/central/baseimage/datastore/tag"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	delegatedRegistryConfigDS "github.com/stackrox/rox/central/delegatedregistryconfig/datastore"
	"github.com/stackrox/rox/central/delegatedregistryconfig/delegator"
	"github.com/stackrox/rox/central/delegatedregistryconfig/scanwaiter"
	"github.com/stackrox/rox/central/delegatedregistryconfig/scanwaiterv2"
	"github.com/stackrox/rox/central/imageintegration"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
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
			repoDS.Singleton(),
			tagDS.Singleton(),
			baseImageDS.Singleton(),
			imageintegration.Set().RegistrySet(),
			scanDelegator,
			env.BaseImageWatcherPollInterval.DurationSetting(),
			env.BaseImageWatcherTagBatchSize.IntegerSetting(),
			env.BaseImageWatcherPerRepoTagLimit.IntegerSetting(),
			features.DelegatedBaseImageScanning.Enabled(),
		)
	})
	return watcher
}
