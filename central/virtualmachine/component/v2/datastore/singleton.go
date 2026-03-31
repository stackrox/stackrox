package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	ds = New(storage)
}

// Singleton returns a singleton instance of the VM component v2 datastore.
func Singleton() DataStore {
	if !features.VirtualMachinesEnhancedDataModel.Enabled() {
		return nil
	}
	once.Do(initialize)
	return ds
}
