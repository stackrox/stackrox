package search

import (
	"context"

	clusterIndexer "github.com/stackrox/rox/central/cluster/index"
	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	clusterSAC "github.com/stackrox/rox/central/cluster/sac"
	"github.com/stackrox/rox/central/componentcveedge/index"
	componentCVEEdgeMappings "github.com/stackrox/rox/central/componentcveedge/mappings"
	"github.com/stackrox/rox/central/componentcveedge/sac"
	"github.com/stackrox/rox/central/componentcveedge/store"
	cveIndexer "github.com/stackrox/rox/central/cve/index"
	cveMappings "github.com/stackrox/rox/central/cve/mappings"
	"github.com/stackrox/rox/central/dackbox"
	deploymentIndexer "github.com/stackrox/rox/central/deployment/index"
	deploymentSAC "github.com/stackrox/rox/central/deployment/sac"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	imageSAC "github.com/stackrox/rox/central/image/sac"
	componentIndexer "github.com/stackrox/rox/central/imagecomponent/index"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	imageComponentEdgeIndexer "github.com/stackrox/rox/central/imagecomponentedge/index"
	imageComponentEdgeMappings "github.com/stackrox/rox/central/imagecomponentedge/mappings"
	imageComponentEdgeSAC "github.com/stackrox/rox/central/imagecomponentedge/sac"
	imageCVEEdgeIndexer "github.com/stackrox/rox/central/imagecveedge/index"
	imageCVEEdgeMappings "github.com/stackrox/rox/central/imagecveedge/mappings"
	imageCVEEdgeSAC "github.com/stackrox/rox/central/imagecveedge/sac"
	nodeIndexer "github.com/stackrox/rox/central/node/index"
	nodeSAC "github.com/stackrox/rox/central/node/sac"
	nodeComponentEdgeIndexer "github.com/stackrox/rox/central/nodecomponentedge/index"
	nodeComponentEdgeMappings "github.com/stackrox/rox/central/nodecomponentedge/mappings"
	nodeComponentEdgeSAC "github.com/stackrox/rox/central/nodecomponentedge/sac"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/compound"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/scoped"
)

type searcherImpl struct {
	storage       store.Store
	indexer       index.Indexer
	graphProvider graph.Provider
	searcher      search.Searcher
}

// SearchComponentCVEEdges returns the search results from indexed cves for the query.
func (ds *searcherImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(ctx, results)
}

// Search returns the raw search results from the query
func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		res, err = ds.searcher.Search(inner, q)
	})
	return res, err
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (count int, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		count, err = ds.searcher.Count(inner, q)
	})
	return count, err
}

// SearchRawComponentCVEEdges retrieves cves from the indexer and storage
func (ds *searcherImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ComponentCVEEdge, error) {
	return ds.searchComponentCVEEdges(ctx, q)
}

// ToComponentCVEEdges returns the cves from the db for the given search results.
func (ds *searcherImpl) resultsToListComponentCVEEdges(ctx context.Context, results []search.Result) ([]*storage.ComponentCVEEdge, []int, error) {
	return ds.storage.GetMany(ctx, search.ResultsToIDs(results))
}

// ToSearchResults returns the searchResults from the db for the given search results.
func (ds *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	cves, missingIndices, err := ds.resultsToListComponentCVEEdges(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(cves, results), nil
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(indexer index.Indexer,
	cveIndexer cveIndexer.Indexer,
	componentIndexer componentIndexer.Indexer,
	imageComponentEdgeIndexer imageComponentEdgeIndexer.Indexer,
	imageCVEEdgeIndexer imageCVEEdgeIndexer.Indexer,
	imageIndexer imageIndexer.Indexer,
	nodeComponentEdgeIndexer nodeComponentEdgeIndexer.Indexer,
	nodeIndexer nodeIndexer.Indexer,
	deploymentIndexer deploymentIndexer.Indexer,
	clusterIndexer clusterIndexer.Indexer) search.Searcher {

	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	componentCVEEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(indexer)
	componentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentIndexer)
	imageComponentEdgeSearcher := filtered.UnsafeSearcher(imageComponentEdgeIndexer, imageComponentEdgeSAC.GetSACFilter())
	imageCVEEdgeSearcher := filtered.UnsafeSearcher(imageCVEEdgeIndexer, imageCVEEdgeSAC.GetSACFilter())
	imageSearcher := filtered.UnsafeSearcher(imageIndexer, imageSAC.GetSACFilter())
	nodeComponentEdgeSearcher := filtered.UnsafeSearcher(nodeComponentEdgeIndexer, nodeComponentEdgeSAC.GetSACFilter())
	nodeSearcher := filtered.UnsafeSearcher(nodeIndexer, nodeSAC.GetSACFilter())
	deploymentSearcher := filtered.UnsafeSearcher(deploymentIndexer, deploymentSAC.GetSACFilter())
	clusterSearcher := filtered.UnsafeSearcher(clusterIndexer, clusterSAC.GetSACFilter())

	compoundSearcher := getCompoundCVESearcher(compoundCVESearcherArg{
		componentCVEEdgeSearcher:   componentCVEEdgeSearcher,
		cveSearcher:                cveSearcher,
		componentSearcher:          componentSearcher,
		imageComponentEdgeSearcher: imageComponentEdgeSearcher,
		imageCVEEdgeSearcher:       imageCVEEdgeSearcher,
		imageSearcher:              imageSearcher,
		nodeComponentEdgeSearcher:  nodeComponentEdgeSearcher,
		nodeSearcher:               nodeSearcher,
		deploymentSearcher:         deploymentSearcher,
		clusterSearcher:            clusterSearcher,
	})
	filteredSearcher := filtered.Searcher(compoundSearcher, sac.GetSACFilter())
	return paginated.Paginated(filteredSearcher)
}

func (ds *searcherImpl) searchComponentCVEEdges(ctx context.Context, q *v1.Query) ([]*storage.ComponentCVEEdge, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	cves, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

type compoundCVESearcherArg struct {
	componentCVEEdgeSearcher   search.Searcher
	cveSearcher                search.Searcher
	componentSearcher          search.Searcher
	imageComponentEdgeSearcher search.Searcher
	imageCVEEdgeSearcher       search.Searcher
	imageSearcher              search.Searcher
	nodeComponentEdgeSearcher  search.Searcher
	nodeSearcher               search.Searcher
	deploymentSearcher         search.Searcher
	clusterSearcher            search.Searcher
}

func getCompoundCVESearcher(arg compoundCVESearcherArg) search.Searcher {

	// The ordering of these is important, so do not change.
	return compound.NewSearcher([]compound.SearcherSpec{
		{
			IsDefault: true,
			Searcher:  scoped.WithScoping(arg.componentCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_COMPONENT_VULN_EDGE)),
			Options:   componentCVEEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.cveSearcher, dackbox.ToCategory(v1.SearchCategory_VULNERABILITIES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_COMPONENT_VULN_EDGE],
			Options:        cveMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.componentSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENTS][v1.SearchCategory_COMPONENT_VULN_EDGE],
			Options:        componentMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.imageComponentEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENT_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENT_EDGE][v1.SearchCategory_COMPONENT_VULN_EDGE],
			Options:        imageComponentEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.imageCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_VULN_EDGE][v1.SearchCategory_COMPONENT_VULN_EDGE],
			Options:        imageCVEEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.imageSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGES][v1.SearchCategory_COMPONENT_VULN_EDGE],
			Options:        dackbox.ImageOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.deploymentSearcher, dackbox.ToCategory(v1.SearchCategory_DEPLOYMENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_DEPLOYMENTS][v1.SearchCategory_COMPONENT_VULN_EDGE],
			Options:        dackbox.DeploymentOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.nodeComponentEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_NODE_COMPONENT_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_NODE_COMPONENT_EDGE][v1.SearchCategory_COMPONENT_VULN_EDGE],
			Options:        nodeComponentEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.nodeSearcher, dackbox.ToCategory(v1.SearchCategory_NODES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_NODES][v1.SearchCategory_COMPONENT_VULN_EDGE],
			Options:        dackbox.NodeOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(arg.clusterSearcher, dackbox.ToCategory(v1.SearchCategory_CLUSTERS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_CLUSTERS][v1.SearchCategory_COMPONENT_VULN_EDGE],
			Options:        clusterMappings.OptionsMap,
		},
	})
}
