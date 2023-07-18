package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	pgStore "github.com/stackrox/rox/central/risk/datastore/internal/store/postgres"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore

	log = logging.LoggerForModule()
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	var err error
	ad, err = New(storage, search.New(storage, pgStore.NewIndexer(globaldb.GetPostgres())))
	if err != nil {
		log.Panicf("Failed to initialize risks datastore: %s", err)
	}
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	once.Do(initialize)
	return ad
}
