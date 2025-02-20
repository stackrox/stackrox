package datastore

import (
	"context"

	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/image/v2/datastore/search"
	"github.com/stackrox/rox/central/cve/image/v2/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to CVE storage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEV2, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ImageCVEV2, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.ImageCVEV2, error)

	EnrichImageWithSuppressedCVEs(image *storage.Image)
}

// New returns a new instance of a DataStore.
func New(storage store.Store, searcher search.Searcher, kf concurrency.KeyFence) DataStore {
	ds := &datastoreImpl{
		storage:  storage,
		searcher: searcher,

		cveSuppressionCache: make(common.CVESuppressionCache),
		keyFence:            kf,
	}
	return ds
}
