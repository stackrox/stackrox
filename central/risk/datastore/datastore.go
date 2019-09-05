package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/risk/datastore/internal/index"
	"github.com/stackrox/rox/central/risk/datastore/internal/search"
	"github.com/stackrox/rox/central/risk/datastore/internal/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// DataStore is an intermediary to RiskStorage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchRawRisks(ctx context.Context, q *v1.Query) ([]*storage.Risk, error)

	GetRisk(ctx context.Context, entityID string, entityType storage.RiskEntityType, aggregateRisk bool) (*storage.Risk, bool, error)
	GetRiskByIndicators(ctx context.Context, entityID string, entityType storage.RiskEntityType, riskIndicatorNames []string) (risk *storage.Risk, err error)
	UpsertRisk(ctx context.Context, risk *storage.Risk) error
	RemoveRisk(ctx context.Context, entityID string, entityType storage.RiskEntityType) error

	GetDependingRiskIDs(riskID string) []string
	GetDependentRiskIDs(riskID string) []string
	AddRiskDependencies(parentRiskID string, dependentIDs ...string)
	RemoveRiskDependencies(riskID string)
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	d := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,

		parentToChildRiskMap: make(map[string]set.StringSet),
		childToParentRiskMap: make(map[string]set.StringSet),
	}
	if err := d.buildIndex(); err != nil {
		return nil, errors.Wrap(err, "failed to build index from existing store")
	}
	d.initRankers()
	d.aggregator = NewRiskAggregator(d)

	return d, nil

}
