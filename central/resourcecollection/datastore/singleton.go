package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ds DataStore
	qr QueryResolver
)

func initialize() {
	var err error
	storage := postgres.New(globaldb.GetPostgres())
	indexer := postgres.NewIndexer(globaldb.GetPostgres())
	ds, qr, err = New(storage, indexer, search.New(storage, indexer))
	utils.CrashOnError(err)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() (DataStore, QueryResolver) {
	if !env.PostgresDatastoreEnabled.BooleanSetting() {
		return nil, nil
	}
	once.Do(initialize)
	return ds, qr
}
