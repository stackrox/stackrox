package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/processindicator"
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/pruner"
	"github.com/stackrox/rox/central/processindicator/search"
	"github.com/stackrox/rox/central/processindicator/store"
	postgresStorage "github.com/stackrox/rox/central/processindicator/store/postgres"
	rocksdbStorage "github.com/stackrox/rox/central/processindicator/store/rocksdb"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/rocksdb"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"gorm.io/gorm"
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
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher, prunerFactory pruner.Factory) (DataStore, error) {
	d := &datastoreImpl{
		storage:               storage,
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

// GetTestPostgresDataStore provides a processindicator datastore hooked on rocksDB and bleve for testing purposes.
func GetTestPostgresDataStore(ctx context.Context, _ *testing.T, pool *pgxpool.Pool, gormDB *gorm.DB) (DataStore, error) {
	dbstore := postgresStorage.CreateTableAndNewStore(ctx, pool, gormDB)
	indexer := postgresStorage.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, indexer, searcher, nil)
}

// GetTestRocksBleveDataStore provides a processindicator datastore hooked on rocksDB and bleve for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, rocksEngine *rocksdb.RocksDB, bleveIndex bleve.Index) (DataStore, error) {
	dbstore := rocksdbStorage.New(rocksEngine)
	indexer := index.New(bleveIndex)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, indexer, searcher, nil)
}
