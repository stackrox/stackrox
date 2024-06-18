package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	store postgres.Store
}

// GetBenchmarksByProfileName returns the benchmarks for the given profile name
func (d datastoreImpl) GetBenchmarksByProfileName(ctx context.Context, profileName string) ([]*storage.ComplianceOperatorBenchmarkV2, error) {
	builder := search.NewQueryBuilder().AddExactMatches(search.ComplianceOperatorProfileName, profileName)
	query := builder.ProtoQuery()

	benchmarks, err := d.store.GetByQuery(ctx, query)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get benchmarks for profile name %s", profileName)
	}
	return benchmarks, nil
}

func (d datastoreImpl) GetBenchmark(ctx context.Context, id string) (*storage.ComplianceOperatorBenchmarkV2, bool, error) {
	return d.store.Get(ctx, id)
}

func (d datastoreImpl) UpsertBenchmark(ctx context.Context, result *storage.ComplianceOperatorBenchmarkV2) error {
	return d.store.Upsert(ctx, result)
}

func (d datastoreImpl) DeleteBenchmark(ctx context.Context, id string) error {
	return d.store.Delete(ctx, id)
}
