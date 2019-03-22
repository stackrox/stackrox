package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/namespace/index"
	"github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	as DataStore
)

func initialize() {
	storage := store.New(globaldb.GetGlobalDB())
	indexer := index.New(globalindex.GetGlobalIndex())

	var err error
	as, err = New(storage, indexer)
	utils.Must(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return as
}
