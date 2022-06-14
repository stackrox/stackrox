package datastore

import (
	"github.com/stackrox/stackrox/central/deployment/cache"
	graphConfigDS "github.com/stackrox/stackrox/central/networkgraph/config/datastore"
	"github.com/stackrox/stackrox/central/networkgraph/entity/networktree"
	"github.com/stackrox/stackrox/central/networkgraph/flow/datastore/internal/store/singleton"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once     sync.Once
	instance ClusterDataStore
)

// Singleton provides the instance of ClusterDataStore to use.
func Singleton() ClusterDataStore {
	once.Do(func() {
		instance = NewClusterDataStore(singleton.Singleton(), graphConfigDS.Singleton(), networktree.Singleton(), cache.DeletedDeploymentCacheSingleton())
	})
	return instance
}
