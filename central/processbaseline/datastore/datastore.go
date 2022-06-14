package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/globaldb"
	"github.com/stackrox/stackrox/central/processbaseline/index"
	"github.com/stackrox/stackrox/central/processbaseline/search"
	"github.com/stackrox/stackrox/central/processbaseline/store"
	"github.com/stackrox/stackrox/central/processbaselineresults/datastore"
	processIndicatorDatastore "github.com/stackrox/stackrox/central/processindicator/datastore"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
	pkgSearch "github.com/stackrox/stackrox/pkg/search"
)

// DataStore wraps storage, indexer, and searcher for ProcessBaselines.
//go:generate mockgen-wrapper
type DataStore interface {
	SearchRawProcessBaselines(ctx context.Context, q *v1.Query) ([]*storage.ProcessBaseline, error)
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)

	GetProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey) (*storage.ProcessBaseline, bool, error)
	AddProcessBaseline(ctx context.Context, baseline *storage.ProcessBaseline) (string, error)
	RemoveProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey) error
	RemoveProcessBaselinesByDeployment(ctx context.Context, deploymentID string) error
	RemoveProcessBaselinesByIDs(ctx context.Context, ids []string) error
	UpdateProcessBaselineElements(ctx context.Context, key *storage.ProcessBaselineKey, addElements []*storage.BaselineItem, removeElements []*storage.BaselineItem, auto bool) (*storage.ProcessBaseline, error)
	UpsertProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey, addElements []*storage.BaselineItem, auto bool, lock bool) (*storage.ProcessBaseline, error)
	UserLockProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey, locked bool) (*storage.ProcessBaseline, error)

	WalkAll(ctx context.Context, fn func(baseline *storage.ProcessBaseline) error) error

	// CreateUnlockedProcessBaseline creates an unlocked baseline
	CreateUnlockedProcessBaseline(ctx context.Context, key *storage.ProcessBaselineKey) (*storage.ProcessBaseline, error)
	// ClearProcessBaselines clears the elements from a process baseline, essentially leaving us with a baseline without processes
	ClearProcessBaselines(ctx context.Context, ids []string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher, processBaselineResults datastore.DataStore, processIndicators processIndicatorDatastore.DataStore) DataStore {
	d := &datastoreImpl{
		storage:                storage,
		indexer:                indexer,
		searcher:               searcher,
		baselineLock:           concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
		processBaselineResults: processBaselineResults,
		processesDataStore:     processIndicators,
	}
	return d
}
