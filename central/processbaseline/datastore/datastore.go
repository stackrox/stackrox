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
	postgresStorage "github.com/stackrox/rox/central/processbaseline/store/postgres"
	rocksdbStorage "github.com/stackrox/rox/central/processbaseline/store/rocksdb"
	"github.com/stackrox/rox/central/processbaselineresults/datastore"
	processIndicatorDatastore "github.com/stackrox/rox/central/processindicator/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/rocksdb"
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

// GetTestPostgresDataStore provides a processbaseline datastore hooked on rocksDB and bleve for testing purposes.
func GetTestPostgresDataStore(ctx context.Context, t *testing.T, pool *pgxpool.Pool, gormDB *gorm.DB) (DataStore, error) {
	dbstore := postgresStorage.CreateTableAndNewStore(ctx, pool, gormDB)
	indexer := postgresStorage.NewIndexer(pool)
	searcher, err := search.New(dbstore, indexer)
	assert.NoError(t, err)
	resultsStore, err := datastore.GetTestPostgresDataStore(ctx, t, pool, gormDB)
	assert.NoError(t, err)
	indicatorStore, err := processIndicatorDatastore.GetTestPostgresDataStore(ctx, t, pool, gormDB)
	assert.NoError(t, err)
	return New(dbstore, indexer, searcher, resultsStore, indicatorStore), nil
}

// GetTestRocksBleveDataStore provides a processbaseline datastore hooked on rocksDB and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksEngine *rocksdb.RocksDB, bleveIndex bleve.Index) (DataStore, error) {
	dbstore, err := rocksdbStorage.New(rocksEngine)
	assert.NoError(t, err)
	indexer := index.New(bleveIndex)
	searcher, err := search.New(dbstore, indexer)
	assert.NoError(t, err)
	resultsStore, err := datastore.GetTestRocksBleveDataStore(t, rocksEngine)
	assert.NoError(t, err)
	indicatorStore, err := processIndicatorDatastore.GetTestRocksBleveDataStore(t, rocksEngine, bleveIndex)
	assert.NoError(t, err)
	return New(dbstore, indexer, searcher, resultsStore, indicatorStore), nil
}
