package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/processbaseline/index"
	"github.com/stackrox/rox/central/processbaseline/search"
	"github.com/stackrox/rox/central/processbaseline/store"
	"github.com/stackrox/rox/central/processbaseline/store/postgres"
	"github.com/stackrox/rox/central/processbaseline/store/rocksdb"
	"github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
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

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(ctx context.Context, t *testing.T, pool *pgxpool.Pool, gormDB *gorm.DB) DataStore {
	postgres.Destroy(ctx, pool)
	dbstore := postgres.CreateTableAndNewStore(ctx, pool, gormDB)
	indexer := postgres.NewIndexer(pool)
	searcher, err := search.New(dbstore, indexer)
	assert.NoError(t, err)
	resultsStore := datastore.GetTestPostgresDataStore(ctx, t, pool, gormDB)
	indicatorStore := processIndicatorDatastore.GetTestPostgresDataStore(ctx, t, pool, gormDB)
	return New(dbstore, indexer, searcher, resultsStore, indicatorStore)
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB, bleveIndex bleve.Index) DataStore {
	dbstore, err := rocksdb.New(rocksengine)
	assert.NoError(t, err)
	indexer := index.New(bleveIndex)
	searcher, err := search.New(dbstore, indexer)
	assert.NoError(t, err)
	resultsStore := datastore.GetTestRocksBleveDataStore(t, rocksengine)
	indicatorStore := processIndicatorDatastore.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	return New(dbstore, indexer, searcher, resultsStore, indicatorStore)
}
