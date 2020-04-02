package datastore

import (
	"context"

	"github.com/blevesearch/bleve"
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/central/pod/datastore/internal/search"
	"github.com/stackrox/rox/central/pod/index"
	badgerStore "github.com/stackrox/rox/central/pod/store/badger"
	"github.com/stackrox/rox/central/pod/store/cache"
	piDS "github.com/stackrox/rox/central/processindicator/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/process/filter"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to PodStorage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchRawPods(ctx context.Context, q *v1.Query) ([]*storage.Pod, error)

	GetPod(ctx context.Context, id string) (*storage.Pod, bool, error)
	GetPods(ctx context.Context, ids []string) ([]*storage.Pod, error)

	UpsertPod(ctx context.Context, pod *storage.Pod) error

	RemovePod(ctx context.Context, id string) error
}

// New creates a pod datastore based on BadgerDB
func New(db *badger.DB, bleveIndex bleve.Index, indicators piDS.DataStore, processFilter filter.Filter) (DataStore, error) {
	store := cache.NewCachedStore(badgerStore.New(db))
	indexer := index.New(bleveIndex)
	searcher := search.New(store, indexer)
	return newDatastoreImpl(store, indexer, searcher, indicators, processFilter)
}
