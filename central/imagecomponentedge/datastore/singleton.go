package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/imagecomponentedge/datastore/internal/store/postgres"
	"github.com/stackrox/rox/central/imagecomponentedge/search"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	var err error
	storage := pgStore.New(globaldb.GetPostgres())
	searcher := search.NewV2(storage, pgStore.NewIndexer(globaldb.GetPostgres()))
	ad, err = New(storage, searcher)
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
