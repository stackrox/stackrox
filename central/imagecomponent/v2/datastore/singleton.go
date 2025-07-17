package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	pgStore "github.com/stackrox/rox/central/imagecomponent/v2/datastore/store/postgres"
	"github.com/stackrox/rox/central/ranking"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	ad DataStore
)

func initialize() {
	storage := pgStore.New(globaldb.GetPostgres())
	ad = New(storage, riskDataStore.Singleton(), ranking.ComponentRanker())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !features.FlattenCVEData.Enabled() {
		return nil
	}
	once.Do(initialize)
	return ad
}
