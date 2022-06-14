package search

import (
	"context"

	clusterMapping "github.com/stackrox/rox/central/cluster/index/mappings"
	clusterSAC "github.com/stackrox/rox/central/cluster/sac"
	componentCVEEdgeMappings "github.com/stackrox/rox/central/componentcveedge/mappings"
	"github.com/stackrox/rox/central/cve/edgefields"
	cveMappings "github.com/stackrox/rox/central/cve/mappings"
	cveSAC "github.com/stackrox/rox/central/cve/sac"
	"github.com/stackrox/rox/central/dackbox"
	deploymentSAC "github.com/stackrox/rox/central/deployment/sac"
	imageSAC "github.com/stackrox/rox/central/image/sac"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	componentSAC "github.com/stackrox/rox/central/imagecomponent/sac"
	"github.com/stackrox/rox/central/imagecomponent/store"
	imageComponentEdgeMappings "github.com/stackrox/rox/central/imagecomponentedge/mappings"
	imageComponentEdgeSAC "github.com/stackrox/rox/central/imagecomponentedge/sac"
	imageCVEEdgeMappings "github.com/stackrox/rox/central/imagecveedge/mappings"
	imageCVEEdgeSAC "github.com/stackrox/rox/central/imagecveedge/sac"
	nodeMappings "github.com/stackrox/rox/central/node/index/mappings"
	nodeSAC "github.com/stackrox/rox/central/node/sac"
	nodeComponentEdgeMappings "github.com/stackrox/rox/central/nodecomponentedge/mappings"
	nodeComponentEdgeSAC "github.com/stackrox/rox/central/nodecomponentedge/sac"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/derivedfields/counter"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/compound"
	"github.com/stackrox/rox/pkg/search/derivedfields"
	"github.com/stackrox/rox/pkg/search/filtered"
	deploymentMappings "github.com/stackrox/rox/pkg/search/options/deployments"
	imageMappings "github.com/stackrox/rox/pkg/search/options/images"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/scoped"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.Component.String(),
	}

	deploymentOnlyOptionsMap = search.Difference(
		deploymentMappings.OptionsMap,
		search.CombineOptionsMaps(
			imageMappings.OptionsMap,
			clusterMapping.OptionsMap,
		),
	)
	imageOnlyOptionsMap = search.Difference(
		imageMappings.OptionsMap,
		search.CombineOptionsMaps(
			imageComponentEdgeMappings.OptionsMap,
			componentMappings.OptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
		),
	)
	nodeOnlyOptionsMap = search.Difference(
		nodeMappings.OptionsMap,
		search.CombineOptionsMaps(
			nodeComponentEdgeMappings.OptionsMap,
			componentMappings.OptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
			clusterMapping.OptionsMap,
		),
	)
)

type searcherImpl struct {
	storage       store.Store
	graphProvider graph.Provider
	searcher      search.Searcher
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.getCountResults(ctx, q)
}

func (ds *searcherImpl) SearchImageComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(ctx, results)
}

func (ds *searcherImpl) SearchRawImageComponents(ctx context.Context, q *v1.Query) ([]*storage.ImageComponent, error) {
	return ds.searchImageComponents(ctx, q)
}

func (ds *searcherImpl) searchImageComponents(ctx context.Context, q *v1.Query) ([]*storage.ImageComponent, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	components, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return components, nil
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		res, err = ds.searcher.Search(inner, q)
	})
	return res, err
}

func (ds *searcherImpl) getCountResults(ctx context.Context, q *v1.Query) (count int, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		count, err = ds.searcher.Count(inner, q)
	})
	return count, err
}

func (ds *searcherImpl) resultsToImageComponents(ctx context.Context, results []search.Result) ([]*storage.ImageComponent, []int, error) {
	return ds.storage.GetMany(ctx, search.ResultsToIDs(results))
}

func (ds *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	components, missingIndices, err := ds.resultsToImageComponents(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(components, results), nil
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(graphProvider graph.Provider,
	cveIndexer blevesearch.UnsafeSearcher,
	componentCVEEdgeIndexer blevesearch.UnsafeSearcher,
	componentIndexer blevesearch.UnsafeSearcher,
	imageComponentEdgeIndexer blevesearch.UnsafeSearcher,
	imageCVEEdgeIndexer blevesearch.UnsafeSearcher,
	imageIndexer blevesearch.UnsafeSearcher,
	nodeComponentEdgeIndexer blevesearch.UnsafeSearcher,
	nodeIndexer blevesearch.UnsafeSearcher,
	deploymentIndexer blevesearch.UnsafeSearcher,
	clusterIndexer blevesearch.UnsafeSearcher,
) search.Searcher {
	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	componentCVEEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentCVEEdgeIndexer)
	componentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentIndexer)
	imageComponentEdgeSearcher := filtered.UnsafeSearcher(imageComponentEdgeIndexer, imageComponentEdgeSAC.GetSACFilter())
	imageCVEEdgeSearcher := filtered.UnsafeSearcher(imageCVEEdgeIndexer, imageCVEEdgeSAC.GetSACFilter())
	imageSearcher := filtered.UnsafeSearcher(imageIndexer, imageSAC.GetSACFilter())
	nodeComponentEdgeSearcher := filtered.UnsafeSearcher(nodeComponentEdgeIndexer, nodeComponentEdgeSAC.GetSACFilter())
	nodeSearcher := filtered.UnsafeSearcher(nodeIndexer, nodeSAC.GetSACFilter())
	deploymentSearcher := filtered.UnsafeSearcher(deploymentIndexer, deploymentSAC.GetSACFilter())
	clusterSearcher := filtered.UnsafeSearcher(clusterIndexer, clusterSAC.GetSACFilter())

	compoundSearcher := getCompoundComponentSearcher(
		cveSearcher,
		componentCVEEdgeSearcher,
		componentSearcher,
		imageComponentEdgeSearcher,
		imageCVEEdgeSearcher,
		imageSearcher,
		nodeComponentEdgeSearcher,
		nodeSearcher,
		deploymentSearcher,
		clusterSearcher)
	filteredSearcher := filtered.Searcher(edgefields.HandleCVEEdgeSearchQuery(compoundSearcher), componentSAC.GetSACFilter())
	// To transform Image to Image Registry, Image Remote, and Image Tag.
	transformedSortSearcher := sortfields.TransformSortFields(filteredSearcher, imageMappings.OptionsMap)
	derivedFieldSortedSearcher := wrapDerivedFieldSearcher(graphProvider, transformedSortSearcher)
	paginatedSearcher := paginated.Paginated(derivedFieldSortedSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}

func getCompoundComponentSearcher(
	cveSearcher search.Searcher,
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
			Searcher:       scoped.WithScoping(cveSearcher, dackbox.ToCategory(v1.SearchCategory_VULNERABILITIES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_IMAGE_COMPONENTS],
			Options:        cveMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(imageCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_VULN_EDGE][v1.SearchCategory_IMAGE_COMPONENTS],
			Options:        imageCVEEdgeMappings.OptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_IMAGE_VULN_EDGE],
		},
		{
			Searcher:       scoped.WithScoping(componentCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_COMPONENT_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_COMPONENT_VULN_EDGE][v1.SearchCategory_IMAGE_COMPONENTS],
			Options:        componentCVEEdgeMappings.OptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_IMAGE_VULN_EDGE][v1.SearchCategory_COMPONENT_VULN_EDGE],
		},
		{
			IsDefault:  true,
			Searcher:   scoped.WithScoping(componentSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENTS)),
			Options:    componentMappings.OptionsMap,
			LinkToPrev: dackbox.GraphTransformations[v1.SearchCategory_COMPONENT_VULN_EDGE][v1.SearchCategory_IMAGE_COMPONENTS],
		},
		{
			Searcher:       scoped.WithScoping(imageComponentEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENT_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENT_EDGE][v1.SearchCategory_IMAGE_COMPONENTS],
			Options:        imageComponentEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(imageSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGES][v1.SearchCategory_IMAGE_COMPONENTS],
			Options:        imageOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(deploymentSearcher, dackbox.ToCategory(v1.SearchCategory_DEPLOYMENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_DEPLOYMENTS][v1.SearchCategory_IMAGE_COMPONENTS],
			Options:        deploymentOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(nodeComponentEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_NODE_COMPONENT_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_NODE_COMPONENT_EDGE][v1.SearchCategory_IMAGE_COMPONENTS],
			Options:        nodeComponentEdgeMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(nodeSearcher, dackbox.ToCategory(v1.SearchCategory_NODES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_NODES][v1.SearchCategory_IMAGE_COMPONENTS],
			Options:        nodeOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(clusterSearcher, dackbox.ToCategory(v1.SearchCategory_CLUSTERS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_CLUSTERS][v1.SearchCategory_IMAGE_COMPONENTS],
			Options:        clusterMapping.OptionsMap,
		},
	})
}

func wrapDerivedFieldSearcher(graphProvider graph.Provider, searcher search.Searcher) search.Searcher {
	return derivedfields.CountSortedSearcher(searcher, map[string]counter.DerivedFieldCounter{
		search.DeploymentCount.String(): counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.ComponentToDeploymentPath, deploymentSAC.GetSACFilter()),
		search.ImageCount.String():      counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.ComponentToImagePath, imageSAC.GetSACFilter()),
		search.CVECount.String():        counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.ComponentToCVEPath, cveSAC.GetSACFilter()),
	})
}
