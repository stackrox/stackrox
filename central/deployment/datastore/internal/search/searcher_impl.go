package search

import (
	"context"

	componentCVEEdgeMappings "github.com/stackrox/stackrox/central/componentcveedge/mappings"
	"github.com/stackrox/stackrox/central/cve/edgefields"
	cveMappings "github.com/stackrox/stackrox/central/cve/mappings"
	cveSAC "github.com/stackrox/stackrox/central/cve/sac"
	"github.com/stackrox/stackrox/central/dackbox"
	"github.com/stackrox/stackrox/central/deployment/index"
	deploymentSAC "github.com/stackrox/stackrox/central/deployment/sac"
	"github.com/stackrox/stackrox/central/deployment/store"
	imageSAC "github.com/stackrox/stackrox/central/image/sac"
	componentMappings "github.com/stackrox/stackrox/central/imagecomponent/mappings"
	imageComponentEdgeMappings "github.com/stackrox/stackrox/central/imagecomponentedge/mappings"
	imageComponentEdgeSAC "github.com/stackrox/stackrox/central/imagecomponentedge/sac"
	imageCVEEdgeMappings "github.com/stackrox/stackrox/central/imagecveedge/mappings"
	imageCVEEdgeSAC "github.com/stackrox/stackrox/central/imagecveedge/sac"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/derivedfields/counter"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
	"github.com/stackrox/stackrox/pkg/search/compound"
	"github.com/stackrox/stackrox/pkg/search/derivedfields"
	"github.com/stackrox/stackrox/pkg/search/filtered"
	deploymentMappings "github.com/stackrox/stackrox/pkg/search/options/deployments"
	imageMappings "github.com/stackrox/stackrox/pkg/search/options/images"
	"github.com/stackrox/stackrox/pkg/search/paginated"
	"github.com/stackrox/stackrox/pkg/search/scoped"
	"github.com/stackrox/stackrox/pkg/search/sortfields"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field:    search.DeploymentPriority.String(),
		Reversed: false,
	}

	componentOptionsMap = search.CombineOptionsMaps(componentMappings.OptionsMap)
	imageOnlyOptionsMap = search.Difference(
		imageMappings.OptionsMap,
		search.CombineOptionsMaps(
			imageComponentEdgeMappings.OptionsMap,
			componentOptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			imageCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
		),
	)
	deploymentOnlyOptionsMap = search.Difference(deploymentMappings.OptionsMap,
		search.CombineOptionsMaps(
			imageOnlyOptionsMap,
			imageComponentEdgeMappings.OptionsMap,
			componentOptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
		),
	)
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage       store.Store
	indexer       index.Indexer
	graphProvider graph.Provider
	searcher      search.Searcher
}

// SearchRawDeployments retrieves deployments from the indexer and storage
func (ds *searcherImpl) SearchRawDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error) {
	deployments, err := ds.searchDeployments(ctx, q)
	if err != nil {
		return nil, err
	}
	return deployments, err
}

// SearchRawDeployments retrieves deployments from the indexer and storage
func (ds *searcherImpl) SearchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, error) {
	deployments, _, err := ds.searchListDeployments(ctx, q)
	if err != nil {
		return nil, err
	}
	return deployments, err
}

func (ds *searcherImpl) searchListDeployments(ctx context.Context, q *v1.Query) ([]*storage.ListDeployment, []search.Result, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	ids := search.ResultsToIDs(results)
	deployments, missingIndices, err := ds.storage.GetManyListDeployments(ctx, ids...)
	if err != nil {
		return nil, nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return deployments, results, nil
}

// SearchDeployments retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchDeployments(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	deployments, results, err := ds.searchListDeployments(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(deployments))
	for i, deployment := range deployments {
		protoResults = append(protoResults, convertDeployment(deployment, results[i]))
	}
	return protoResults, nil
}

func (ds *searcherImpl) searchDeployments(ctx context.Context, q *v1.Query) ([]*storage.Deployment, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	deployments, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		res, err = ds.searcher.Search(inner, q)
	})
	return res, err
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (res int, err error) {
	graph.Context(ctx, ds.graphProvider, func(inner context.Context) {
		res, err = ds.searcher.Count(inner, q)
	})
	return res, err
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(graphProvider graph.Provider,
	cveIndexer blevesearch.UnsafeSearcher,
	componentCVEEdgeIndexer blevesearch.UnsafeSearcher,
	componentIndexer blevesearch.UnsafeSearcher,
	imageComponentEdgeIndexer blevesearch.UnsafeSearcher,
	imageIndexer blevesearch.UnsafeSearcher,
	deploymentIndexer blevesearch.UnsafeSearcher,
	imageCVEEdgeIndexer blevesearch.UnsafeSearcher) search.Searcher {

	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	componentCVEEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentCVEEdgeIndexer)
	componentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentIndexer)
	imageComponentEdgeSearcher := filtered.UnsafeSearcher(imageComponentEdgeIndexer, imageComponentEdgeSAC.GetSACFilter())
	imageSearcher := filtered.UnsafeSearcher(imageIndexer, imageSAC.GetSACFilter())
	deploymentSearcher := filtered.UnsafeSearcher(deploymentIndexer, deploymentSAC.GetSACFilter())
	imageCVEEdgeSearcher := filtered.UnsafeSearcher(imageCVEEdgeIndexer, imageCVEEdgeSAC.GetSACFilter())

	compoundSearcher := getDeploymentCompoundSearcher(
		cveSearcher,
		componentCVEEdgeSearcher,
		componentSearcher,
		imageComponentEdgeSearcher,
		imageSearcher,
		deploymentSearcher,
		imageCVEEdgeSearcher)
	filteredSearcher := filtered.Searcher(edgefields.HandleCVEEdgeSearchQuery(compoundSearcher), deploymentSAC.GetSACFilter()) // Make the UnsafeSearcher safe.
	// To transform Image to Image Registry, Image Remote, and Image Tag.
	transformedSortFieldSearcher := sortfields.TransformSortFields(filteredSearcher, deploymentMappings.OptionsMap)
	derivedFieldSortedSearcher := wrapDerivedFieldSearcher(graphProvider, transformedSortFieldSearcher)
	paginatedSearcher := paginated.Paginated(derivedFieldSortedSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}

func getDeploymentCompoundSearcher(
	cveSearcher search.Searcher,
	componentCVEEdgeSearcher search.Searcher,
	componentSearcher search.Searcher,
	imageComponentEdgeSearcher search.Searcher,
	imageSearcher search.Searcher,
	deploymentSearcher search.Searcher,
	imageCVEEdgeSearcher search.Searcher) search.Searcher {
	// The ordering of these is important, so do not change.
	return compound.NewSearcher([]compound.SearcherSpec{
		{
			Searcher:       scoped.WithScoping(cveSearcher, dackbox.ToCategory(v1.SearchCategory_VULNERABILITIES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_DEPLOYMENTS],
			Options:        cveMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(componentCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_COMPONENT_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_COMPONENT_VULN_EDGE][v1.SearchCategory_DEPLOYMENTS],
			Options:        componentCVEEdgeMappings.OptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_COMPONENT_VULN_EDGE],
		},
		{
			Searcher:       scoped.WithScoping(componentSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENTS][v1.SearchCategory_DEPLOYMENTS],
			Options:        componentOptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_COMPONENT_VULN_EDGE][v1.SearchCategory_IMAGE_COMPONENTS],
		},
		{
			Searcher:       scoped.WithScoping(imageComponentEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENT_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENT_EDGE][v1.SearchCategory_DEPLOYMENTS],
			Options:        imageComponentEdgeMappings.OptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENTS][v1.SearchCategory_IMAGE_COMPONENT_EDGE],
		},
		{
			Searcher:       scoped.WithScoping(imageSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGES][v1.SearchCategory_DEPLOYMENTS],
			Options:        imageOnlyOptionsMap,
		},
		{
			IsDefault: true,
			Searcher:  scoped.WithScoping(deploymentSearcher, dackbox.ToCategory(v1.SearchCategory_DEPLOYMENTS)),
			Options:   deploymentOnlyOptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(imageCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_VULN_EDGE][v1.SearchCategory_DEPLOYMENTS],
			Options:        imageCVEEdgeMappings.OptionsMap,
		},
	})
}

func wrapDerivedFieldSearcher(graphProvider graph.Provider, searcher search.Searcher) search.Searcher {
	return derivedfields.CountSortedSearcher(searcher, map[string]counter.DerivedFieldCounter{
		search.ImageCount.String(): counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.DeploymentToImage, imageSAC.GetSACFilter()),
		search.CVECount.String():   counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.DeploymentToCVE, cveSAC.GetSACFilter()),
	})
}
