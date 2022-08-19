package datastore

import (
	"context"
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/central/imageintegration/index"
	"github.com/stackrox/rox/central/imageintegration/search"
	"github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/imageintegration/store/bolt"
	"github.com/stackrox/rox/central/imageintegration/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"go.etcd.io/bbolt"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
//go:generate mockgen-wrapper
type DataStore interface {
	GetImageIntegration(ctx context.Context, id string) (*storage.ImageIntegration, bool, error)
	GetImageIntegrations(ctx context.Context, integration *v1.GetImageIntegrationsRequest) ([]*storage.ImageIntegration, error)

	AddImageIntegration(ctx context.Context, integration *storage.ImageIntegration) (string, error)
	UpdateImageIntegration(ctx context.Context, integration *storage.ImageIntegration) error
	RemoveImageIntegration(ctx context.Context, id string) error
	Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error)
	SearchImageIntegrations(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
}

// New returns an instance of DataStore.
func New(imageIntegrationStorage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:           imageIntegrationStorage,
		indexer:           indexer,
		formattedSearcher: searcher,
	}

	ctx := sac.WithAllAccess(context.Background())

	if err := ds.buildIndex(ctx); err != nil {
		log.Fatal("unable to load search index for image integrations")
	}
	return ds
}

// NewForTestOnly returns an instance of DataStore only for tests.
func NewForTestOnly(imageIntegrationStorage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:           imageIntegrationStorage,
		indexer:           indexer,
		formattedSearcher: searcher,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(t *testing.T, pool *pgxpool.Pool) (DataStore, error) {
	store := postgres.New(pool)
	indexer := postgres.NewIndexer(pool)
	searcher := search.New(store, indexer)
	return New(store, indexer, searcher), nil
}

// GetTestRocksBleveDataStore provides a datastore connected to rocksdb and bleve for testing purposes.
func GetTestRocksBleveDataStore(_ *testing.T, boltengine *bbolt.DB, bleveIndex bleve.Index) (DataStore, error) {
	testStore := bolt.New(boltengine)
	indexer := index.New(bleveIndex)
	searcher := search.New(testStore, indexer)
	return New(testStore, indexer, searcher), nil
}
