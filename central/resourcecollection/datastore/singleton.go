package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/resourcecollection/datastore/search"
	pgStore "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
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
	storage := pgStore.New(globaldb.GetPostgres())
	ds, qr, err = New(storage, search.New(storage, pgStore.NewIndexer(globaldb.GetPostgres())))
	utils.CrashOnError(err)
}

// Singleton returns a singleton instance of cve datastore
func Singleton() (DataStore, QueryResolver) {
	once.Do(initialize)
	return ds, qr
}
