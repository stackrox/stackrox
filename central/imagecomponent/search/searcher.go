package search

import (
	"context"

	componentCVEEdgeIndexer "github.com/stackrox/rox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	"github.com/stackrox/rox/central/imagecomponent/store"
	imageComponentEdgeIndexer "github.com/stackrox/rox/central/imagecomponentedge/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/search"
)

// Searcher provides search functionality on existing image components.
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	SearchImageComponents(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImageComponents(ctx context.Context, query *v1.Query) ([]*storage.ImageComponent, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, graphProvider graph.Provider,
	cveIndexer cveIndexer.Indexer,
	componentCVEEdgeIndexer componentCVEEdgeIndexer.Indexer,
	componentIndexer componentIndexer.Indexer,
	imageComponentEdgeIndexer imageComponentEdgeIndexer.Indexer,
	imageIndexer imageIndexer.Indexer,
	deploymentIndexer deploymentIndexer.Indexer) Searcher {
	return &searcherImpl{
		storage:       storage,
		graphProvider: graphProvider,
		searcher: formatSearcher(graphProvider,
			cveIndexer,
			componentCVEEdgeIndexer,
			componentIndexer,
			imageComponentEdgeIndexer,
			imageIndexer,
			deploymentIndexer),
	}
}
