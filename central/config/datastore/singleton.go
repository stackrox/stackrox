package datastore

import (
	"github.com/stackrox/rox/central/config/store"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	d DataStore
)

func initialize() {
	d = New(store.New(globaldb.GetGlobalDB()))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return d
}
