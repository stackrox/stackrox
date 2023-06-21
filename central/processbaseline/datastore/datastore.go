package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/processbaseline/search"
	"github.com/stackrox/rox/central/processbaseline/store"
	pgStore "github.com/stackrox/rox/central/processbaseline/store/postgres"
	"github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/require"
)

// DataStore wraps storage, and searcher for ProcessBaselines.
//
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

// New returns a new instance of DataStore using the input store, and searcher.
func New(storage store.Store, searcher search.Searcher, processBaselineResults datastore.DataStore, processIndicators processIndicatorDatastore.DataStore) DataStore {
	d := &datastoreImpl{
		storage:                storage,
		searcher:               searcher,
		baselineLock:           concurrency.NewKeyedMutex(globaldb.DefaultDataStorePoolSize),
		processBaselineResults: processBaselineResults,
		processesDataStore:     processIndicators,
	}
	return d
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t testing.TB, pool postgres.DB) (DataStore, error) {
	dbStore := pgStore.New(pool)
	store, err := pgStore.NewWithCache(dbStore)
	require.NoError(t, err)
	indexer := pgStore.NewIndexer(pool)
	searcher, err := search.New(store, indexer)
	if err != nil {
		return nil, err
	}
	resultsStore, err := datastore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	indicatorStore, err := processIndicatorDatastore.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	return New(store, searcher, resultsStore, indicatorStore), nil
}
