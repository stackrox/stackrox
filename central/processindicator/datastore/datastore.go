package datastore

import (
	"context"

	"github.com/stackrox/rox/central/analystnotes"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/internal/commentsstore"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	pkgSearch "github.com/stackrox/rox/pkg/search"
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

	// Comments-related methods.
	AddProcessComment(ctx context.Context, processKey *analystnotes.ProcessNoteKey, comment *storage.Comment) (string, error)
	UpdateProcessComment(ctx context.Context, processKey *analystnotes.ProcessNoteKey, comment *storage.Comment) error
	GetCommentsForProcess(ctx context.Context, processKey *analystnotes.ProcessNoteKey) ([]*storage.Comment, error)
	GetCommentsCountForProcess(ctx context.Context, processKey *analystnotes.ProcessNoteKey) (int, error)
	RemoveProcessComment(ctx context.Context, processKey *analystnotes.ProcessNoteKey, commentID string) error

	WalkAll(ctx context.Context, fn func(pi *storage.ProcessIndicator) error) error

	// Stop signals all goroutines associated with this object to terminate.
	Stop() bool
	// Wait waits until all goroutines associated with this object have terminated, or cancelWhen gets triggered.
	// A return value of false indicates that cancelWhen was triggered.
	Wait(cancelWhen concurrency.Waitable) bool
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, commentsStorage commentsstore.Store, indexer index.Indexer, searcher search.Searcher, prunerFactory pruner.Factory) (DataStore, error) {
	d := &datastoreImpl{
		storage:               storage,
		commentsStorage:       commentsStorage,
		indexer:               indexer,
		searcher:              searcher,
		prunerFactory:         prunerFactory,
		prunedArgsLengthCache: make(map[processindicator.ProcessWithContainerInfo]int),
		stopSig:               concurrency.NewSignal(),
		stoppedSig:            concurrency.NewSignal(),
	}
	ctx := context.TODO()
	if err := d.buildIndex(ctx); err != nil {
		return nil, err
	}
	go d.prunePeriodically(ctx)
	return d, nil
}
