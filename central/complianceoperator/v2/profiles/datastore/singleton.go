package datastore

import (
	"github.com/stackrox/rox/central/complianceoperator/v2/profiles/datastore/search"
	edge "github.com/stackrox/rox/central/complianceoperator/v2/profiles/profileclusteredge/store/postgres"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/profiles/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	dataStore DataStore
)

func initialize() {
	db := globaldb.GetPostgres()
	indexer := pgStore.NewIndexer(db)
	storage := pgStore.New(db)
	profileSearch := search.New(storage, indexer)

	dataStore = New(
		storage,
		edge.New(db),
		db,
		profileSearch,
	)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return dataStore
}
