package datastore

import (
	"context"

	"github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/store/postgres"
	"github.com/stackrox/rox/generated/storage"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	store postgres.Store
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
