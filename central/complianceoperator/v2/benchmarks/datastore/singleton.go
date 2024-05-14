package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/pkg/defaults/complianceoperator"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	dataStore DataStore
)

func initialize() {
	storage := postgres.New(globaldb.GetPostgres())
	dataStore = New(storage)

	setupBenchmarks(storage)
}

// Singleton provides the interface for non-service external interaction.
func Singleton() DataStore {
	if !features.ComplianceEnhancements.Enabled() {
		return nil
	}
	once.Do(initialize)
	return dataStore
}

func setupBenchmarks(s postgres.Store) {
	benchmarks, err := complianceoperator.LoadComplianceOperatorBenchmarks()
	utils.CrashOnError(err)

	for _, b := range benchmarks {
		if err := s.Upsert(context.TODO(), b); err != nil {
			utils.Must(err)
		}
	}
}
