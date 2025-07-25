package datastore

import (
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
	storage := pgStore.New(db)

	dataStore = New(
		storage,
		db,
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
