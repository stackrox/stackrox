package datastore

import (
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton returns the singleton providing access to the product usage store.
func Singleton() DataStore {
	once.Do(func() {
		ds = New(clusterDS.Singleton())
	})
	return ds
}
