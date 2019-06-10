package datastore

import (
	"context"

	"github.com/blevesearch/bleve"
	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/image/datastore/internal/search"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	"github.com/stackrox/rox/central/image/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to AlertStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	SearchListImages(ctx context.Context, q *v1.Query) ([]*storage.ListImage, error)
	ListImage(ctx context.Context, sha string) (*storage.ListImage, bool, error)
	ListImages(ctx context.Context) ([]*storage.ListImage, error)

	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.Image, error)

	GetImages(ctx context.Context) ([]*storage.Image, error)
	CountImages(ctx context.Context) (int, error)
	GetImage(ctx context.Context, sha string) (*storage.Image, bool, error)
	GetImagesBatch(ctx context.Context, shas []string) ([]*storage.Image, error)
	UpsertImage(ctx context.Context, image *storage.Image) error

	DeleteImages(ctx context.Context, ids ...string) error
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
// noUpdateTimestamps controls whether timestamps are automatically updated when upserting images.
// This should be set to `false` except for some tests.
func New(db *bbolt.DB, bleveIndex bleve.Index, noUpdateTimestamps bool) (DataStore, error) {
	storage := store.New(db, noUpdateTimestamps)
	indexer := index.New(bleveIndex)
	searcher := search.New(storage, indexer)
	return newDatastoreImpl(storage, indexer, searcher)
}
