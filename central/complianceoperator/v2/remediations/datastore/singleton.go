package datastore

import (
	"github.com/stackrox/rox/central/complianceoperator/v2/remediations/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	dataStore DataStore
)

func initialize() {
	storage := postgres.New(globaldb.GetPostgres())
	dataStore = New(storage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !features.ComplianceEnhancements.Enabled() && !features.ComplianceRemediationV2.Enabled() {
		return nil
	}
	once.Do(initialize)
	return dataStore
}
