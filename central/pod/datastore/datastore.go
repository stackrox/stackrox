package datastore

import (
	"context"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/pod/datastore/internal/search"
	"github.com/stackrox/rox/central/pod/index"
	"github.com/stackrox/rox/central/pod/store/cache"
	"github.com/stackrox/rox/central/pod/store/postgres"
	"github.com/stackrox/rox/central/pod/store/rocksdb"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
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
	store := cache.NewCachedStore(rocksdb.New(db))
	indexer := index.New(bleveIndex)
	searcher := search.New(store, indexer)
	return newDatastoreImpl(context.TODO(), store, indexer, searcher, indicators, processFilter)
}

// NewPostgresDB creates a pod datastore based on Postgres
func NewPostgresDB(db *pgxpool.Pool, indicators piDS.DataStore, processFilter filter.Filter) (DataStore, error) {
	store := cache.NewCachedStore(postgres.New(db))
	indexer := postgres.NewIndexer(db)
	searcher := search.New(store, indexer)
	return newDatastoreImpl(context.TODO(), store, indexer, searcher, indicators, processFilter)
}
