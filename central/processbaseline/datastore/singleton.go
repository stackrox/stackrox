package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/processbaseline/search"
	pgStore "github.com/stackrox/rox/central/processbaseline/store/postgres"
	"github.com/stackrox/rox/central/processbaselineresults/datastore"
	indicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())

	searcher, err := search.New(storage)
	if err != nil {
		panic("unable to load search index for process baseline")
	}

	ad = New(storage, searcher, datastore.Singleton(), indicatorStore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
