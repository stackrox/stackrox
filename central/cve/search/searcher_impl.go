package search

import (
	"context"

	clusterMappings "github.com/stackrox/stackrox/central/cluster/index/mappings"
	clusterSAC "github.com/stackrox/stackrox/central/cluster/sac"
	clusterCVEEdgeMappings "github.com/stackrox/stackrox/central/clustercveedge/mappings"
	clusterCVEEdgeSAC "github.com/stackrox/stackrox/central/clustercveedge/sac"
	componentCVEEdgeMappings "github.com/stackrox/stackrox/central/componentcveedge/mappings"
	"github.com/stackrox/stackrox/central/cve/edgefields"
	"github.com/stackrox/stackrox/central/cve/index"
	cveMappings "github.com/stackrox/stackrox/central/cve/mappings"
	cveSAC "github.com/stackrox/stackrox/central/cve/sac"
	"github.com/stackrox/stackrox/central/cve/store"
	"github.com/stackrox/stackrox/central/dackbox"
	deploymentSAC "github.com/stackrox/stackrox/central/deployment/sac"
	imageSAC "github.com/stackrox/stackrox/central/image/sac"
	componentMappings "github.com/stackrox/stackrox/central/imagecomponent/mappings"
	componentSAC "github.com/stackrox/stackrox/central/imagecomponent/sac"
	imageComponentEdgeMappings "github.com/stackrox/stackrox/central/imagecomponentedge/mappings"
	imageComponentEdgeSAC "github.com/stackrox/stackrox/central/imagecomponentedge/sac"
	imageCVEEdgeMappings "github.com/stackrox/stackrox/central/imagecveedge/mappings"
	imageCVEEdgeSAC "github.com/stackrox/stackrox/central/imagecveedge/sac"
	nodeSAC "github.com/stackrox/stackrox/central/node/sac"
	nodeComponentEdgeMappings "github.com/stackrox/stackrox/central/nodecomponentedge/mappings"
	nodeComponentEdgeSAC "github.com/stackrox/stackrox/central/nodecomponentedge/sac"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/derivedfields/counter"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
	"github.com/stackrox/stackrox/pkg/search/compound"
	"github.com/stackrox/stackrox/pkg/search/derivedfields"
	"github.com/stackrox/stackrox/pkg/search/filtered"
	"github.com/stackrox/stackrox/pkg/search/paginated"
	"github.com/stackrox/stackrox/pkg/search/scoped"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.CVE.String(),
	}
)

type searcherImpl struct {
	storage       store.Store
	indexer       index.Indexer
	graphProvider graph.Provider
	searcher      search.Searcher
}

func (ds *searcherImpl) SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(ctx, results)
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.getCount(ctx, q)
}

func (ds *searcherImpl) SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error) {
	return ds.searchCVEs(ctx, q)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		res, err = ds.searcher.Search(inner, q)
	})
	return res, err
}

func (ds *searcherImpl) getCount(ctx context.Context, q *v1.Query) (count int, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		count, err = ds.searcher.Count(inner, q)
	})
	return count, err
}

func (ds *searcherImpl) resultsToCVEs(ctx context.Context, results []search.Result) ([]*storage.CVE, []int, error) {
	return ds.storage.GetMany(ctx, search.ResultsToIDs(results))
}

func (ds *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	cves, missingIndices, err := ds.resultsToCVEs(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(cves, results), nil
}

func convertMany(cves []*storage.CVE, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(cve *storage.CVE, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_VULNERABILITIES,
		Id:             cve.GetId(),
		Name:           cve.GetId(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(graphProvider graph.Provider,
	cveIndexer blevesearch.UnsafeSearcher,
	clusterCVEEdgeIndexer blevesearch.UnsafeSearcher,
	componentCVEEdgeIndexer blevesearch.UnsafeSearcher,
	componentIndexer blevesearch.UnsafeSearcher,
	imageComponentEdgeIndexer blevesearch.UnsafeSearcher,
	imageCVEEdgeIndexer blevesearch.UnsafeSearcher,
	imageIndexer blevesearch.UnsafeSearcher,
	nodeComponentEdgeIndexer blevesearch.UnsafeSearcher,
	nodeIndexer blevesearch.UnsafeSearcher,
	deploymentIndexer blevesearch.UnsafeSearcher,
	clusterIndexer blevesearch.UnsafeSearcher) search.Searcher {

	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	clusterCVEEdgeSearcher := filtered.UnsafeSearcher(clusterCVEEdgeIndexer, clusterCVEEdgeSAC.GetSACFilter())
	componentCVEEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentCVEEdgeIndexer)
	componentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentIndexer)
	imageComponentEdgeSearcher := filtered.UnsafeSearcher(imageComponentEdgeIndexer, imageComponentEdgeSAC.GetSACFilter())
	imageCVEEdgeSearcher := filtered.UnsafeSearcher(imageCVEEdgeIndexer, imageCVEEdgeSAC.GetSACFilter())
	imageSearcher := filtered.UnsafeSearcher(imageIndexer, imageSAC.GetSACFilter())
	nodeComponentEdgeSearcher := filtered.UnsafeSearcher(nodeComponentEdgeIndexer, nodeComponentEdgeSAC.GetSACFilter())
	nodeSearcher := filtered.UnsafeSearcher(nodeIndexer, nodeSAC.GetSACFilter())
	deploymentSearcher := filtered.UnsafeSearcher(deploymentIndexer, deploymentSAC.GetSACFilter())
	clusterSearcher := filtered.UnsafeSearcher(clusterIndexer, clusterSAC.GetSACFilter())

	compoundSearcher := getCompoundCVESearcher(
		cveSearcher,
		clusterCVEEdgeSearcher,
		componentCVEEdgeSearcher,
		componentSearcher,
		imageComponentEdgeSearcher,
		imageCVEEdgeSearcher,
		imageSearcher,
		nodeComponentEdgeSearcher,
		nodeSearcher,
		deploymentSearcher,
		clusterSearcher)
	filteredSearcher := filtered.Searcher(
		edgefields.HandleSnoozeSearchQuery(edgefields.HandleCVEEdgeSearchQuery(compoundSearcher)),
		cveSAC.GetSACFilter())
	derivedFieldSortedSearcher := wrapDerivedFieldSearcher(graphProvider, filteredSearcher)
	paginatedSearcher := paginated.Paginated(derivedFieldSortedSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}

func (ds *searcherImpl) searchCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error) {
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

func getCompoundCVESearcher(
	cveSearcher search.Searcher,
	clusterCVEEdgeSearcher search.Searcher,
	componentCVEEdgeSearcher search.Searcher,
	componentSearcher search.Searcher,
	imageComponentEdgeSearcher search.Searcher,
	imageCVEEdgeSearcher search.Searcher,
	imageSearcher search.Searcher,
	nodeComponentEdgeSearcher search.Searcher,
	nodeSearcher search.Searcher,
	deploymentSearcher search.Searcher,
	clusterSearcher search.Searcher) search.Searcher {
	// The ordering of these is important, so do not change.
	return compound.NewSearcher([]compound.SearcherSpec{
		{
			IsDefault: true,
			Searcher:  scoped.WithScoping(cveSearcher, dackbox.ToCategory(v1.SearchCategory_VULNERABILITIES)),
			Options:   cveMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(componentCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_COMPONENT_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_COMPONENT_VULN_EDGE][v1.SearchCategory_VULNERABILITIES],
			Options:        componentCVEEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(clusterCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_CLUSTER_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_CLUSTER_VULN_EDGE][v1.SearchCategory_VULNERABILITIES],
			Options:        clusterCVEEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(componentSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENTS][v1.SearchCategory_VULNERABILITIES],
			Options:        componentMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(imageComponentEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENT_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENT_EDGE][v1.SearchCategory_VULNERABILITIES],
			Options:        imageComponentEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(imageCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_VULN_EDGE][v1.SearchCategory_VULNERABILITIES],
			Options:        imageCVEEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(imageSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGES][v1.SearchCategory_VULNERABILITIES],
			Options:        dackbox.ImageOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(deploymentSearcher, dackbox.ToCategory(v1.SearchCategory_DEPLOYMENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_DEPLOYMENTS][v1.SearchCategory_VULNERABILITIES],
			Options:        dackbox.DeploymentOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(nodeComponentEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_NODE_COMPONENT_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_NODE_COMPONENT_EDGE][v1.SearchCategory_VULNERABILITIES],
			Options:        nodeComponentEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(nodeSearcher, dackbox.ToCategory(v1.SearchCategory_NODES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_NODES][v1.SearchCategory_VULNERABILITIES],
			Options:        dackbox.NodeOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(clusterSearcher, dackbox.ToCategory(v1.SearchCategory_CLUSTERS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_CLUSTERS][v1.SearchCategory_VULNERABILITIES],
			Options:        clusterMappings.OptionsMap,
		},
	})
}

func wrapDerivedFieldSearcher(graphProvider graph.Provider, searcher search.Searcher) search.Searcher {
	return derivedfields.CountSortedSearcher(searcher, map[string]counter.DerivedFieldCounter{
		search.DeploymentCount.String(): counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.CVEToDeploymentPath, deploymentSAC.GetSACFilter()),
		search.ImageCount.String():      counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.CVEToImagePath, imageSAC.GetSACFilter()),
		search.NodeCount.String():       counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.CVEToNodePath, nodeSAC.GetSACFilter()),
		search.ComponentCount.String():  counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.CVEToComponentPath, componentSAC.GetSACFilter()),
	})
}
