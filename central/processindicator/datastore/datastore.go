package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/processindicator"
	"github.com/stackrox/stackrox/central/processindicator/index"
	"github.com/stackrox/stackrox/central/processindicator/pruner"
	"github.com/stackrox/stackrox/central/processindicator/search"
	"github.com/stackrox/stackrox/central/processindicator/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/sac"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
)

// DataStore represents the interface to access data.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchRawProcessIndicators(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error)

	GetProcessIndicator(ctx context.Context, id string) (*storage.ProcessIndicator, bool, error)
	GetProcessIndicators(ctx context.Context, ids []string) ([]*storage.ProcessIndicator, bool, error)
	AddProcessIndicators(context.Context, ...*storage.ProcessIndicator) error
	RemoveProcessIndicatorsByPod(ctx context.Context, id string) error
	RemoveProcessIndicators(ctx context.Context, ids []string) error

	WalkAll(ctx context.Context, fn func(pi *storage.ProcessIndicator) error) error

	// Stop signals all goroutines associated with this object to terminate.
	Stop() bool
	// Wait waits until all goroutines associated with this object have terminated, or cancelWhen gets triggered.
	// A return value of false indicates that cancelWhen was triggered.
	Wait(cancelWhen concurrency.Waitable) bool
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(store store.Store, indexer index.Indexer, searcher search.Searcher, prunerFactory pruner.Factory) (DataStore, error) {
	d := &datastoreImpl{
		storage:               store,
		indexer:               indexer,
		searcher:              searcher,
		prunerFactory:         prunerFactory,
		prunedArgsLengthCache: make(map[processindicator.ProcessWithContainerInfo]int),
		stopSig:               concurrency.NewSignal(),
		stoppedSig:            concurrency.NewSignal(),
	}
	ctx := sac.WithAllAccess(context.Background())
	if err := d.buildIndex(ctx); err != nil {
		return nil, err
	}
	go d.prunePeriodically(ctx)
	return d, nil
}
