package datastore

import (
	statusStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/scanconfigstatus/store/postgres"
	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	dataStore DataStore
)

func initialize() {
	pool := globaldb.GetPostgres()
	storage := pgStore.New(pool)
	dataStore = New(storage, statusStore.New(pool), pool)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return dataStore
}
