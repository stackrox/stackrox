package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/nodecomponentcveedge/datastore/search"
	pgStore "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/store/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ds DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())

	ds = New(storage, search.New(storage, pgStore.NewIndexer(globaldb.GetPostgres())))
}

// Singleton returns a singleton instance of cve datastore
func Singleton() DataStore {
	once.Do(initialize)
	return ds
}
