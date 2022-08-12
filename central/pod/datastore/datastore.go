package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/pod/datastore/internal/search"
	"github.com/stackrox/rox/central/pod/index"
	"github.com/stackrox/rox/central/pod/store/cache"
	"github.com/stackrox/rox/central/pod/store/postgres"
	"github.com/stackrox/rox/central/pod/store/rocksdb"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	piFilter "github.com/stackrox/rox/central/processindicator/filter"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/process/filter"
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to PodStorage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchRawPods(ctx context.Context, q *v1.Query) ([]*storage.Pod, error)

	GetPod(ctx context.Context, id string) (*storage.Pod, bool, error)
	WalkAll(ctx context.Context, fn func(pod *storage.Pod) error) error

	UpsertPod(ctx context.Context, pod *storage.Pod) error

	RemovePod(ctx context.Context, id string) error

	GetPodIDs(ctx context.Context) ([]string, error)
}

// NewRocksDB creates a pod datastore based on RocksDB
func NewRocksDB(db *rocksdbBase.RocksDB, bleveIndex bleve.Index, indicators piDS.DataStore, processFilter filter.Filter) (DataStore, error) {
	store, err := cache.NewCachedStore(rocksdb.New(db))
	if err != nil {
		return nil, err
	}
	indexer := index.New(bleveIndex)
	searcher := search.New(store, indexer)
	return newDatastoreImpl(context.TODO(), store, indexer, searcher, indicators, processFilter)
}

// NewPostgresDB creates a pod datastore based on Postgres
func NewPostgresDB(db *pgxpool.Pool, indicators piDS.DataStore, processFilter filter.Filter) (DataStore, error) {
	store, err := cache.NewCachedStore(postgres.New(db))
	if err != nil {
		return nil, err
	}
	indexer := postgres.NewIndexer(db)
	searcher := search.New(store, indexer)
	return newDatastoreImpl(context.TODO(), store, indexer, searcher, indicators, processFilter)
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	processIndicatorStore, err := piDS.GetTestPostgresDataStore(t, pool)
	if err != nil {
		return nil, err
	}
	processIndicatorFilter := piFilter.Singleton()
	return NewPostgresDB(pool, processIndicatorStore, processIndicatorFilter)
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB, bleveIndex bleve.Index) (DataStore, error) {
	processIndicatorStore, err := piDS.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	if err != nil {
		return nil, err
	}
	processIndicatorFilter := piFilter.Singleton()
	return NewRocksDB(rocksengine, bleveIndex, processIndicatorStore, processIndicatorFilter)
}
