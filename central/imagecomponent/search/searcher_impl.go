package search

import (
	"context"

	componentCVEEdgeMappings "github.com/stackrox/rox/central/componentcveedge/mappings"
	cveMappings "github.com/stackrox/rox/central/cve/mappings"
	cveSAC "github.com/stackrox/rox/central/cve/sac"
	"github.com/stackrox/rox/central/dackbox"
	deploymentSAC "github.com/stackrox/rox/central/deployment/sac"
	imageSAC "github.com/stackrox/rox/central/image/sac"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	componentSAC "github.com/stackrox/rox/central/imagecomponent/sac"
	"github.com/stackrox/rox/central/imagecomponent/store"
	imageComponentEdgeMappings "github.com/stackrox/rox/central/imagecomponentedge/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/derivedfields/counter"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/compound"
	"github.com/stackrox/rox/pkg/search/derivedfields"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/search/idspace"
	deploymentMappings "github.com/stackrox/rox/pkg/search/options/deployments"
	imageMappings "github.com/stackrox/rox/pkg/search/options/images"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.Component.String(),
	}

	deploymentOnlyOptionsMap = search.Difference(deploymentMappings.OptionsMap, imageMappings.ImageOnlyOptionsMap)
	imageOnlyOptionsMap      = search.Difference(search.Difference(imageMappings.ImageOnlyOptionsMap, cveMappings.OptionsMap), componentMappings.OptionsMap)
)

type searcherImpl struct {
	storage  store.Store
	searcher search.Searcher
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

func (ds *searcherImpl) SearchImageComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(results)
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
	components, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return nil, err
	}
	return components, nil
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *searcherImpl) resultsToImageComponents(results []search.Result) ([]*storage.ImageComponent, []int, error) {
	return ds.storage.GetBatch(search.ResultsToIDs(results))
}

func (ds *searcherImpl) resultsToSearchResults(results []search.Result) ([]*v1.SearchResult, error) {
	components, missingIndices, err := ds.resultsToImageComponents(results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(components, results), nil
}

func convertMany(components []*storage.ImageComponent, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(components))
	for index, sar := range components {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(component *storage.ImageComponent, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGE_COMPONENTS,
		Id:             component.GetId(),
		Name:           component.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(graphProvider idspace.GraphProvider,
	cveIndexer blevesearch.UnsafeSearcher,
	componentCVEEdgeIndexer blevesearch.UnsafeSearcher,
	componentIndexer blevesearch.UnsafeSearcher,
	imageComponentEdgeIndexer blevesearch.UnsafeSearcher,
	imageIndexer blevesearch.UnsafeSearcher,
	deploymentIndexer blevesearch.UnsafeSearcher) search.Searcher {
	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	componentCVEEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentCVEEdgeIndexer)
	componentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentIndexer)
	imageComponentEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(imageComponentEdgeIndexer)
	imageSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(imageIndexer)
	deploymentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(deploymentIndexer)

	compoundSearcher := getCompoundComponentSearcher(graphProvider,
		cveSearcher,
		componentCVEEdgeSearcher,
		componentSearcher,
		imageComponentEdgeSearcher,
		imageSearcher,
		deploymentSearcher)
	filteredSearcher := filtered.Searcher(compoundSearcher, componentSAC.GetSACFilter(graphProvider))
	transformedSortSearcher := sortfields.TransformSortFields(filteredSearcher)
	derivedFieldSortedSearcher := wrapDerivedFieldSearcher(graphProvider, transformedSortSearcher)
	paginatedSearcher := paginated.Paginated(derivedFieldSortedSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}

func getCompoundComponentSearcher(graphProvider idspace.GraphProvider,
	cveSearcher search.Searcher,
	componentCVEEdgeSearcher search.Searcher,
	componentSearcher search.Searcher,
	imageComponentEdgeSearcher search.Searcher,
	imageSearcher search.Searcher,
	deploymentSearcher search.Searcher) search.Searcher {

	return compound.NewSearcher([]compound.SearcherSpec{
		{
			Searcher: idspace.TransformIDs(cveSearcher, idspace.NewBackwardGraphTransformer(graphProvider, dackbox.CVEToComponentPath.Path)),
			Options:  cveMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(componentCVEEdgeSearcher, idspace.NewEdgeToParentTransformer()),
			Options:  componentCVEEdgeMappings.OptionsMap,
		},
		{
			IsDefault: true,
			Searcher:  componentSearcher,
			Options:   componentMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(imageComponentEdgeSearcher, idspace.NewEdgeToChildTransformer()),
			Options:  imageComponentEdgeMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(imageSearcher, idspace.NewForwardGraphTransformer(graphProvider, dackbox.ImageToComponentPath.Path)),
			Options:  imageOnlyOptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(deploymentSearcher, idspace.NewForwardGraphTransformer(graphProvider, dackbox.DeploymentToImageComponent.Path)),
			Options:  deploymentOnlyOptionsMap,
		},
	}...)
}

func wrapDerivedFieldSearcher(graphProvider graph.Provider, searcher search.Searcher) search.Searcher {
	if !features.Dackbox.Enabled() {
		return searcher
	}
	return derivedfields.CountSortedSearcher(searcher, map[string]counter.DerivedFieldCounter{
		search.DeploymentCount.String(): counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.ComponentToDeploymentPath, deploymentSAC.GetSACFilter(graphProvider)),
		search.ImageCount.String():      counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.ComponentToImagePath, imageSAC.GetSACFilter(graphProvider)),
		search.CVECount.String():        counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.ComponentToCVEPath, cveSAC.GetSACFilters(graphProvider)...),
	})
}
