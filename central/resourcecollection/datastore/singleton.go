package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := postgres.New(globaldb.GetPostgres())
	indexer := postgres.NewIndexer(globaldb.GetPostgres())
	ds = New(storage, indexer, search.New(storage, indexer))
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	if !features.ObjectCollections.Enabled() {
		return nil
	}
	once.Do(initialize)
	return ds
}
