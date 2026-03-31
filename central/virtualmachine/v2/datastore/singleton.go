package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/virtualmachine/v2/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	store := pgStore.New(globaldb.GetPostgres(), concurrency.NewKeyFence())
	ds = newDatastoreImpl(store)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !features.VirtualMachinesEnhancedDataModel.Enabled() {
		return nil
	}
	once.Do(initialize)
	return ds
}
