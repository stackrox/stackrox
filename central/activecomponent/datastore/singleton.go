package datastore

import (
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store/dackbox"
	globaldb "github.com/stackrox/rox/central/globaldb/dackbox"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := dackbox.New(globaldb.GetGlobalDackBox())
	ds = New(globaldb.GetGlobalDackBox(), storage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
