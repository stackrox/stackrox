package datastore

import (
	"context"

	"github.com/stackrox/rox/central/imageintegration/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
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
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
}

// New returns an instance of DataStore.
func New(storage store.Store, searcher search.Searcher) DataStore {
	return &datastoreImpl{
		storage:           storage,
		formattedSearcher: searcher,
	}
}
