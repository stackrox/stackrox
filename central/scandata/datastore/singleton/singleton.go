package singleton

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/scandata/datastore"
	"github.com/stackrox/rox/central/scandata/datastore/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds datastore.DataStore
)

// Singleton provides the interface for non-service external interaction.
func Singleton() datastore.DataStore {
	once.Do(func() {
		ds = postgres.New(globaldb.GetPostgres())
	})
	return ds
}
