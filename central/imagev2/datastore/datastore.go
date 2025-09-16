package datastore

import (
	"context"

	"github.com/stackrox/rox/central/imagev2/datastore/store"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to ImageV2Storage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	SearchImages(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImages(ctx context.Context, q *v1.Query) ([]*storage.ImageV2, error)

	GetImage(ctx context.Context, id string) (*storage.ImageV2, bool, error)
	GetImageMetadata(ctx context.Context, id string) (*storage.ImageV2, bool, error)
	GetManyImageMetadata(ctx context.Context, ids []string) ([]*storage.ImageV2, error)
	GetImagesBatch(ctx context.Context, ids []string) ([]*storage.ImageV2, error)
	WalkByQuery(ctx context.Context, q *v1.Query, fn func(image *storage.ImageV2) error) error

	UpsertImage(ctx context.Context, image *storage.ImageV2) error
	UpdateVulnerabilityState(ctx context.Context, cve string, images []string, state storage.VulnerabilityState) error

	DeleteImages(ctx context.Context, ids ...string) error
	Exists(ctx context.Context, id string) (bool, error)
}

// NewWithPostgres returns a new instance of DataStore using the input store, and searcher.
func NewWithPostgres(storage store.Store, risks riskDS.DataStore, imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) DataStore {
	ds := newDatastoreImpl(storage, risks, imageRanker, imageComponentRanker)
	go ds.initializeRankers()
	return ds
}
