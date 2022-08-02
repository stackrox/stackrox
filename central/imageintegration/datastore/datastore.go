package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/imageintegration/index"
	"github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	genSt "github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/testutils"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
//go:generate mockgen-wrapper
type DataStore interface {
	GetImageIntegration(ctx context.Context, id string) (*genSt.ImageIntegration, bool, error)
	GetImageIntegrations(ctx context.Context, integration *v1.GetImageIntegrationsRequest) ([]*genSt.ImageIntegration, error)

	AddImageIntegration(ctx context.Context, integration *genSt.ImageIntegration) (string, error)
	UpdateImageIntegration(ctx context.Context, integration *genSt.ImageIntegration) error
	RemoveImageIntegration(ctx context.Context, id string) error
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
}

// New returns an instance of DataStore.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	ds := &datastoreImpl{
		storage:           storage,
		indexer:           indexer,
		formattedSearcher: searcher,
	}

	ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(genSt.Access_READ_ACCESS, genSt.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ImageIntegration)))

	if err := ds.buildIndex(ctx); err != nil {
		log.Error(err.Error())
		return nil
	}
	return ds
}

// NewForTestOnly returns an instance of DataStore only for tests.
func NewForTestOnly(t *testing.T, storage store.Store, indexer index.Indexer, searcher search.Searcher) DataStore {
	testutils.MustBeInTest(t)
	ds := &datastoreImpl{
		storage:           storage,
		indexer:           indexer,
		formattedSearcher: searcher,
	}
	return ds
}
