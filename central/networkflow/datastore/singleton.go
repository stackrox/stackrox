package datastore

import (
	"github.com/stackrox/rox/central/networkflow/datastore/internal/store/singleton"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance ClusterDataStore
)

// Singleton provides the instance of ClusterDataStore to use.
func Singleton() ClusterDataStore {
	once.Do(func() {
		instance = NewClusterDataStore(singleton.Singleton())
	})
	return instance
}
