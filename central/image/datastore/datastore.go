package datastore

import (
	"context"

	"github.com/stackrox/rox/central/image/datastore/search"
	"github.com/stackrox/rox/central/image/datastore/store"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to AlertStorage.
//
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
	GetManyImageMetadata(ctx context.Context, ids []string) ([]*storage.Image, error)
	GetImagesBatch(ctx context.Context, shas []string) ([]*storage.Image, error)

	UpsertImage(ctx context.Context, image *storage.Image) error
	UpdateVulnerabilityState(ctx context.Context, cve string, images []string, state storage.VulnerabilityState) error

	DeleteImages(ctx context.Context, ids ...string) error
	Exists(ctx context.Context, id string) (bool, error)
}

// NewWithPostgres returns a new instance of DataStore using the input store, and searcher.
// noUpdateTimestamps controls whether timestamps are automatically updated when upserting images.
// This should be set to `false` except for some tests.
func NewWithPostgres(storage store.Store, index imageIndexer.Indexer, risks riskDS.DataStore, imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) DataStore {
	ds := newDatastoreImpl(storage, search.NewV2(storage, index), risks, imageRanker, imageComponentRanker)
	ds.initializeRankers()
	return ds
}
