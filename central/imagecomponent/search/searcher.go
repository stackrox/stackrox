package search

import (
	"context"

	clusterIndexer "github.com/stackrox/stackrox/central/cluster/index"
	componentCVEEdgeIndexer "github.com/stackrox/stackrox/central/componentcveedge/index"
	cveIndexer "github.com/stackrox/stackrox/central/cve/index"
	deploymentIndexer "github.com/stackrox/stackrox/central/deployment/index"
	imageIndexer "github.com/stackrox/stackrox/central/image/index"
	componentIndexer "github.com/stackrox/stackrox/central/imagecomponent/index"
	"github.com/stackrox/stackrox/central/imagecomponent/store"
	imageComponentEdgeIndexer "github.com/stackrox/stackrox/central/imagecomponentedge/index"
	imageCVEEdgeIndexer "github.com/stackrox/stackrox/central/imagecveedge/index"
	nodeIndexer "github.com/stackrox/stackrox/central/node/index"
	nodeComponentEdgeIndexer "github.com/stackrox/stackrox/central/nodecomponentedge/index"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/search"
)

// Searcher provides search functionality on existing image components.
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchImageComponents(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawImageComponents(ctx context.Context, query *v1.Query) ([]*storage.ImageComponent, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, graphProvider graph.Provider,
	cveIndexer cveIndexer.Indexer,
	componentCVEEdgeIndexer componentCVEEdgeIndexer.Indexer,
	componentIndexer componentIndexer.Indexer,
	imageComponentEdgeIndexer imageComponentEdgeIndexer.Indexer,
	imageCVEEdgeIndexer imageCVEEdgeIndexer.Indexer,
	imageIndexer imageIndexer.Indexer,
	nodeComponentEdgeIndexer nodeComponentEdgeIndexer.Indexer,
	nodeIndexer nodeIndexer.Indexer,
	deploymentIndexer deploymentIndexer.Indexer,
	clusterIndexer clusterIndexer.Indexer) Searcher {
	return &searcherImpl{
		storage:       storage,
		graphProvider: graphProvider,
		searcher: formatSearcher(graphProvider,
			cveIndexer,
			componentCVEEdgeIndexer,
			componentIndexer,
			imageComponentEdgeIndexer,
			imageCVEEdgeIndexer,
			imageIndexer,
			nodeComponentEdgeIndexer,
			nodeIndexer,
			deploymentIndexer,
			clusterIndexer),
	}
}
