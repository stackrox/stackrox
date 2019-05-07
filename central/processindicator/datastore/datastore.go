package datastore

import (
	"context"

	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to PolicyStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchRawProcessIndicators(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error)

	GetProcessIndicator(ctx context.Context, id string) (*storage.ProcessIndicator, bool, error)
	GetProcessIndicators(ctx context.Context) ([]*storage.ProcessIndicator, error)
	AddProcessIndicator(context.Context, *storage.ProcessIndicator) error
	AddProcessIndicators(context.Context, ...*storage.ProcessIndicator) error
	RemoveProcessIndicator(ctx context.Context, id string) error
	RemoveProcessIndicatorsByDeployment(ctx context.Context, id string) error
	RemoveProcessIndicatorsOfStaleContainers(ctx context.Context, deploymentID string, currentContainerIDs []string) error

	// Stop signals all goroutines associated with this object to terminate.
	Stop() bool
	// Wait waits until all goroutines associated with this object have terminated, or cancelWhen gets triggered.
	// A return value of false indicates that cancelWhen was triggered.
	Wait(cancelWhen concurrency.Waitable) bool
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher, prunerFactory pruner.Factory) DataStore {
	d := &datastoreImpl{
		storage:       storage,
		indexer:       indexer,
		searcher:      searcher,
		prunerFactory: prunerFactory,
		stopSig:       concurrency.NewSignal(),
		stoppedSig:    concurrency.NewSignal(),
	}
	go d.prunePeriodically()
	return d
}
