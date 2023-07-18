package datastore

import (
	pgStore "github.com/stackrox/rox/central/clustercveedge/datastore/store/postgres"
	"github.com/stackrox/rox/central/clustercveedge/search"
	"github.com/stackrox/rox/central/globaldb"
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
	ad, err = New(storage, search.NewV2(storage, pgStore.NewIndexer(globaldb.GetPostgres())))
	utils.CrashOnError(err)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
