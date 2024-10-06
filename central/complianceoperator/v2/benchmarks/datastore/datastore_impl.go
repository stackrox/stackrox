package datastore

import (
	"context"

	"github.com/pkg/errors"
	searcher "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/search"
	"github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var _ DataStore = (*datastoreImpl)(nil)

type datastoreImpl struct {
	store    postgres.Store
	searcher searcher.Searcher
}

func (d datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return d.searcher.Search(ctx, q)
}

func (d datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return d.searcher.Count(ctx, q)
}

func (d datastoreImpl) SearchBenchmarks(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return d.searcher.SearchBenchmarks(ctx, q)
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
