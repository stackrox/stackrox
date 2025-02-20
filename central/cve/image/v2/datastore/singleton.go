package datastore

import (
	"github.com/stackrox/rox/central/cve/image/v2/datastore/search"
	pgStore "github.com/stackrox/rox/central/cve/image/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())

	ds = New(storage, search.New(storage))
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	if !features.FlattenCVEData.Enabled() {
		return nil
	}
	once.Do(initialize)
	return ds
}
