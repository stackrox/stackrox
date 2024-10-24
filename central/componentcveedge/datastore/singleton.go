package datastore

import (
	pgStore "github.com/stackrox/rox/central/componentcveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/componentcveedge/search"
	"github.com/stackrox/rox/central/componentcveedge/store"
	"github.com/stackrox/rox/central/globaldb"
	pg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	searcher := search.NewV2(storage)
	ad = New(storage, searcher)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}

func NewStorage(db pg.DB) store.Store {
	return pgStore.New(db)
}
