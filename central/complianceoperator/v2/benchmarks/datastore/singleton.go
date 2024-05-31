package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/store/postgres"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/complianceoperator"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	once sync.Once

	dataStore DataStore

	complianceOperatorBenchmarkAdministrationCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Compliance)))
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
	utils.Should(errors.Wrap(err, "Unable to load the compliance operator benchmarks"))

	for _, b := range benchmarks {
		if err := s.Upsert(complianceOperatorBenchmarkAdministrationCtx, b); err != nil {
			utils.Should(errors.Wrapf(err, "Unable to upsert the compliance operator benchmark %s", b.GetName()))
		}
	}
}
