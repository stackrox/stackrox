package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/imageintegration/search"
	"github.com/stackrox/rox/central/imageintegration/store"
	pgStore "github.com/stackrox/rox/central/imageintegration/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
	pkgSearch "github.com/stackrox/rox/pkg/search"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
//
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
func New(imageIntegrationStorage store.Store, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:           imageIntegrationStorage,
		formattedSearcher: searcher,
	}
	return ds
}

// NewForTestOnly returns an instance of DataStore only for tests.
func NewForTestOnly(imageIntegrationStorage store.Store, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:           imageIntegrationStorage,
		formattedSearcher: searcher,
	}
	return ds
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ *testing.T, pool postgres.DB) (DataStore, error) {
	store := pgStore.New(pool)
	searcher := search.New(store, pgStore.NewIndexer(pool))
	return New(store, searcher), nil
}
