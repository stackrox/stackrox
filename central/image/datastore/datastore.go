package datastore

import (
	"context"

	"github.com/blevesearch/bleve"
	"github.com/dgraph-io/badger"
	componentCVEEdgeIndexer "github.com/stackrox/rox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/image/datastore/internal/search"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	badgerStore "github.com/stackrox/rox/central/image/datastore/internal/store/badger"
	dackBoxStore "github.com/stackrox/rox/central/image/datastore/internal/store/dackbox"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/datastore"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/rox/central/imagecomponentedge/index"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	"github.com/stackrox/rox/pkg/features"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to AlertStorage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	SearchListImages(ctx context.Context, q *v1.Query) ([]*storage.ListImage, error)
	ListImage(ctx context.Context, sha string) (*storage.ListImage, bool, error)

	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.Image, error)

	CountImages(ctx context.Context) (int, error)
	GetImage(ctx context.Context, sha string) (*storage.Image, bool, error)
	GetImagesBatch(ctx context.Context, shas []string) ([]*storage.Image, error)

	UpsertImage(ctx context.Context, image *storage.Image) error

	DeleteImages(ctx context.Context, ids ...string) error
	Exists(ctx context.Context, id string) (bool, error)
}

func newDatastore(dacky *dackbox.DackBox, storage store.Store, bleveIndex bleve.Index, noUpdateTimestamps bool,
	imageComponents imageComponentDS.DataStore, risks riskDS.DataStore, imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) (DataStore, error) {
	var searcher search.Searcher
	indexer := imageIndexer.New(bleveIndex)

	if features.Dackbox.Enabled() {
		searcher = search.New(storage,
			dacky,
			cveIndexer.New(bleveIndex),
			componentCVEEdgeIndexer.New(bleveIndex),
			componentIndexer.New(bleveIndex),
			imageComponentEdgeIndexer.New(bleveIndex),
			imageIndexer.New(bleveIndex))
	} else {
		searcher = search.New(storage, nil, nil, nil, nil, nil, indexer)
	}

	ds, err := newDatastoreImpl(storage, indexer, searcher, imageComponents, risks, imageRanker, imageComponentRanker)
	if err != nil {
		return nil, err
	}

	if features.Dackbox.Enabled() {
		ds.initializeRankers()
	}
	return ds, nil
}

// NewBadger returns a new instance of DataStore using the input store, indexer, and searcher.
// noUpdateTimestamps controls whether timestamps are automatically updated when upserting images.
// This should be set to `false` except for some tests.
func NewBadger(dacky *dackbox.DackBox, keyFence concurrency.KeyFence, db *badger.DB, bleveIndex bleve.Index, noUpdateTimestamps bool,
	imageComponents imageComponentDS.DataStore, risks riskDS.DataStore, imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) (DataStore, error) {
	var storage store.Store
	if features.Dackbox.Enabled() {
		var err error
		storage, err = dackBoxStore.New(dacky, keyFence, noUpdateTimestamps)
		if err != nil {
			return nil, err
		}
	} else {
		storage = badgerStore.New(db, noUpdateTimestamps)
	}
	return newDatastore(dacky, storage, bleveIndex, noUpdateTimestamps, imageComponents, risks, imageRanker, imageComponentRanker)
}
