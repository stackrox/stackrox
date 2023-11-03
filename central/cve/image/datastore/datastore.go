package datastore

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/image/datastore/search"
	"github.com/stackrox/rox/central/cve/image/datastore/store"
	pgStore "github.com/stackrox/rox/central/cve/image/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to CVE storage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchImageCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImageCVEs(ctx context.Context, q *v1.Query) ([]*storage.ImageCVE, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.ImageCVE, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.ImageCVE, error)

	Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, cves ...string) error
	Unsuppress(ctx context.Context, cves ...string) error

	// ApplyException and RevertException are used for database backward compatibility purpose only. They can be removed after a few releases.
	ApplyException(ctx context.Context, start *types.Timestamp, expiry *types.Timestamp, cves ...string) error
	RevertException(ctx context.Context, cves ...string) error

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
	ds.buildSuppressedCache()
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) DataStore {
	dbstore := pgStore.New(pool)
	indexer := pgStore.NewIndexer(pool)
	searcher := search.New(dbstore, indexer)
	return New(dbstore, searcher, concurrency.NewKeyFence())
}
