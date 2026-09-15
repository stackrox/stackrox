package datastore

import (
	pgStore "github.com/stackrox/rox/central/aiworkload/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	ad   DataStore
)

func initialize() {
	store := pgStore.New(globaldb.GetPostgres())
	ad = newDatastoreImpl(store)
}

func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
