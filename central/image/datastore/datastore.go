package datastore

import (
	"context"

	"github.com/blevesearch/bleve"
	componentCVEEdgeIndexer "github.com/stackrox/rox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/image/datastore/internal/search"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	dackBoxStore "github.com/stackrox/rox/central/image/datastore/internal/store/dackbox"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageCVEEdgeIndexer "github.com/stackrox/rox/central/imagecveedge/index"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/dackbox"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to AlertStorage.
//go:generate mockgen-wrapper
type DataStore interface {
	SearchListImages(ctx context.Context, q *v1.Query) ([]*storage.ListImage, error)
	ListImage(ctx context.Context, sha string) (*storage.ListImage, bool, error)

	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.Image, error)

	CountImages(ctx context.Context) (int, error)
	GetImage(ctx context.Context, sha string) (*storage.Image, bool, error)
	GetImageMetadata(ctx context.Context, id string) (*storage.Image, bool, error)
	GetImagesBatch(ctx context.Context, shas []string) ([]*storage.Image, error)

	UpsertImage(ctx context.Context, image *storage.Image) error

	DeleteImages(ctx context.Context, ids ...string) error
	Exists(ctx context.Context, id string) (bool, error)
}

func newDatastore(dacky *dackbox.DackBox, storage store.Store, bleveIndex bleve.Index, processIndex bleve.Index, risks riskDS.DataStore, imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) DataStore {
	indexer := imageIndexer.New(bleveIndex)

	searcher := search.New(storage,
		dacky,
		cveIndexer.New(bleveIndex),
		componentCVEEdgeIndexer.New(bleveIndex),
		componentIndexer.New(bleveIndex),
		imageComponentEdgeIndexer.New(bleveIndex),
		imageIndexer.New(bleveIndex),
		deploymentIndexer.New(bleveIndex, processIndex),
		imageCVEEdgeIndexer.New(bleveIndex),
	)
	ds := newDatastoreImpl(storage, indexer, searcher, risks, imageRanker, imageComponentRanker)
	ds.initializeRankers()

	return ds
}

// New returns a new instance of DataStore using the input store, indexer, and searcher.
// noUpdateTimestamps controls whether timestamps are automatically updated when upserting images.
// This should be set to `false` except for some tests.
func New(dacky *dackbox.DackBox, keyFence concurrency.KeyFence, bleveIndex bleve.Index, processIndex bleve.Index, noUpdateTimestamps bool, risks riskDS.DataStore, imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) DataStore {
	storage := dackBoxStore.New(dacky, keyFence, noUpdateTimestamps)
	return newDatastore(dacky, storage, bleveIndex, processIndex, risks, imageRanker, imageComponentRanker)
}

// NewWithPostgres returns a new instance of DataStore using the input store, indexer, and searcher.
// noUpdateTimestamps controls whether timestamps are automatically updated when upserting images.
// This should be set to `false` except for some tests.
func NewWithPostgres(storage store.Store, index imageIndexer.Indexer, noUpdateTimestamps bool, risks riskDS.DataStore, imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) DataStore {
	ds := newDatastoreImpl(storage, index, search.NewV2(storage, index), risks, imageRanker, imageComponentRanker)
	ds.initializeRankers()
	return ds
}
