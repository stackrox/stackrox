package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/jackc/pgx/v4/pgxpool"
	componentCVEEdgeIndexer "github.com/stackrox/rox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/image/datastore/internal/search"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	dackBoxStore "github.com/stackrox/rox/central/image/datastore/internal/store/dackbox"
	postgresStore "github.com/stackrox/rox/central/image/datastore/internal/store/postgres"
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
	rocksdbBase "github.com/stackrox/rox/pkg/rocksdb"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"gorm.io/gorm"
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
	UpdateVulnerabilityState(ctx context.Context, cve string, images []string, state storage.VulnerabilityState) error

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
func NewWithPostgres(storage store.Store, index imageIndexer.Indexer, risks riskDS.DataStore, imageRanker *ranking.Ranker, imageComponentRanker *ranking.Ranker) DataStore {
	ds := newDatastoreImpl(storage, index, search.NewV2(storage, index), risks, imageRanker, imageComponentRanker)
	ds.initializeRankers()
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(ctx context.Context, t *testing.T, pool *pgxpool.Pool, gormDB *gorm.DB) DataStore {
	postgresStore.Destroy(ctx, pool)
	dbstore := postgresStore.CreateTableAndNewStore(ctx, pool, gormDB)
	indexer := postgresStore.NewIndexer(pool)
	riskStore := riskDS.GetTestPostgresDataStore(ctx, t, pool, gormDB)
	imageRanker := ranking.ImageRanker()
	imageComponentRanker := ranking.ComponentRanker()
	return NewWithPostgres(dbstore, indexer, riskStore, imageRanker, imageComponentRanker)
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(t *testing.T, rocksengine *rocksdbBase.RocksDB, bleveIndex bleve.Index, dacky *dackbox.DackBox, keyFence concurrency.KeyFence) DataStore {
	riskStore := riskDS.GetTestRocksBleveDataStore(t, rocksengine, bleveIndex)
	imageRanker := ranking.ImageRanker()
	imageComponentRanker := ranking.ComponentRanker()
	return New(dacky, keyFence, bleveIndex, bleveIndex, false, riskStore, imageRanker, imageComponentRanker)
}
