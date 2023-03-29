package search

import (
	"context"

	acIndexer "github.com/stackrox/rox/central/activecomponent/datastore/index"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on active components
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, q *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, q *v1.Query) (int, error)

	SearchRawActiveComponents(ctx context.Context, q *v1.Query) ([]*storage.ActiveComponent, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store,
	graphProvider graph.Provider,
	acIndexer acIndexer.Indexer,
	cveIndexer cveIndexer.Indexer,
	componentIndexer componentIndexer.Indexer,
	deploymentIndexer deploymentIndexer.Indexer) Searcher {
	return &searcherImpl{
		storage:       storage,
		graphProvider: graphProvider,
		searcher:      formatSearcher(acIndexer, cveIndexer, componentIndexer, deploymentIndexer),
	}
}
