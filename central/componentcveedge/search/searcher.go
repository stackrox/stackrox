package search

import (
	"context"

	clusterIndexer "github.com/stackrox/stackrox/central/cluster/index"
	"github.com/stackrox/stackrox/central/componentcveedge/index"
	"github.com/stackrox/stackrox/central/componentcveedge/store"
	cveIndexer "github.com/stackrox/stackrox/central/cve/index"
	deploymentIndexer "github.com/stackrox/stackrox/central/deployment/index"
	imageIndexer "github.com/stackrox/stackrox/central/image/index"
	componentIndexer "github.com/stackrox/stackrox/central/imagecomponent/index"
	imageComponentEdgeIndexer "github.com/stackrox/stackrox/central/imagecomponentedge/index"
	imageCVEEdgeIndexer "github.com/stackrox/stackrox/central/imagecveedge/index"
	nodeIndexer "github.com/stackrox/stackrox/central/node/index"
	nodeComponentEdgeIndexer "github.com/stackrox/stackrox/central/nodecomponentedge/index"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/search"
)

// Searcher provides search functionality on existing cves.
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchEdges(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, query *v1.Query) ([]*storage.ComponentCVEEdge, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, graphProvider graph.Provider,
	indexer index.Indexer,
	cveIndexer cveIndexer.Indexer,
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
		indexer:       indexer,
		graphProvider: graphProvider,
		searcher: formatSearcher(
			indexer,
			cveIndexer,
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
