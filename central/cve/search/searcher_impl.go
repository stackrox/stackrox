package search

import (
	"context"

	clusterMappings "github.com/stackrox/rox/central/cluster/index/mappings"
	clusterCVEEdgeMappings "github.com/stackrox/rox/central/clustercveedge/mappings"
	componentCVEEdgeMappings "github.com/stackrox/rox/central/componentcveedge/mappings"
	"github.com/stackrox/rox/central/cve/index"
	cveMappings "github.com/stackrox/rox/central/cve/mappings"
	cveSAC "github.com/stackrox/rox/central/cve/sac"
	"github.com/stackrox/rox/central/cve/store"
	"github.com/stackrox/rox/central/dackbox"
	deploymentSAC "github.com/stackrox/rox/central/deployment/sac"
	imageSAC "github.com/stackrox/rox/central/image/sac"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	componentSAC "github.com/stackrox/rox/central/imagecomponent/sac"
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
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.CVE.String(),
	}

	deploymentOnlyOptionsMap = search.Difference(deploymentMappings.OptionsMap, imageMappings.OptionsMap)
)

type searcherImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (ds *searcherImpl) SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return ds.resultsToSearchResults(results)
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.getSearchResults(ctx, q)
}

func (ds *searcherImpl) SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error) {
	return ds.searchCVEs(ctx, q)
}

func (ds *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *searcherImpl) resultsToCVEs(results []search.Result) ([]*storage.CVE, []int, error) {
	return ds.storage.GetBatch(search.ResultsToIDs(results))
}

func (ds *searcherImpl) resultsToSearchResults(results []search.Result) ([]*v1.SearchResult, error) {
	cves, missingIndices, err := ds.resultsToCVEs(results)
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
func formatSearcher(graphProvider idspace.GraphProvider,
	cveIndexer blevesearch.UnsafeSearcher,
	clusterCVEEdgeIndexer blevesearch.UnsafeSearcher,
	componentCVEEdgeIndexer blevesearch.UnsafeSearcher,
	componentIndexer blevesearch.UnsafeSearcher,
	imageComponentEdgeIndexer blevesearch.UnsafeSearcher,
	imageIndexer blevesearch.UnsafeSearcher,
	deploymentIndexer blevesearch.UnsafeSearcher,
	clusterIndexer blevesearch.UnsafeSearcher) search.Searcher {

	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	clusterCVEEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(clusterCVEEdgeIndexer)
	componentCVEEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentCVEEdgeIndexer)
	componentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentIndexer)
	imageComponentEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(imageComponentEdgeIndexer)
	imageSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(imageIndexer)
	deploymentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(deploymentIndexer)
	clusterSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(clusterIndexer)

	compoundSearcher := getCompoundCVESearcher(graphProvider,
		cveSearcher,
		clusterCVEEdgeSearcher,
		componentCVEEdgeSearcher,
		componentSearcher,
		imageComponentEdgeSearcher,
		imageSearcher,
		deploymentSearcher,
		clusterSearcher)
	filteredSearcher := filtered.Searcher(compoundSearcher, cveSAC.GetSACFilters(graphProvider)...)
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
	cves, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

func getCompoundCVESearcher(graphProvider idspace.GraphProvider,
	cveSearcher search.Searcher,
	clusterCVEEdgeSearcher search.Searcher,
	componentCVEEdgeSearcher search.Searcher,
	componentSearcher search.Searcher,
	imageComponentEdgeSearcher search.Searcher,
	imageSearcher search.Searcher,
	deploymentSearcher search.Searcher,
	clusterSearcher search.Searcher) search.Searcher {
	imageComponentEdgeToComponentSearcher := idspace.TransformIDs(imageComponentEdgeSearcher, idspace.NewEdgeToChildTransformer())
	return compound.NewSearcher([]compound.SearcherSpec{
		{
			IsDefault: true,
			Searcher:  cveSearcher,
			Options:   cveMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(componentCVEEdgeSearcher, idspace.NewEdgeToChildTransformer()),
			Options:  componentCVEEdgeMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(clusterCVEEdgeSearcher, idspace.NewEdgeToChildTransformer()),
			Options:  clusterCVEEdgeMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(componentSearcher, idspace.NewForwardGraphTransformer(graphProvider, dackbox.ComponentToCVEPath.Path)),
			Options:  componentMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(imageComponentEdgeToComponentSearcher, idspace.NewForwardGraphTransformer(graphProvider, dackbox.ComponentToCVEPath.Path)),
			Options:  imageComponentEdgeMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(clusterSearcher, idspace.NewForwardGraphTransformer(graphProvider, dackbox.ClusterToCVE.Path, dackbox.ClusterToClusterCVE.Path)),
			Options:  clusterMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(deploymentSearcher, idspace.NewForwardGraphTransformer(graphProvider, dackbox.DeploymentToCVE.Path)),
			Options:  deploymentOnlyOptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(imageSearcher, idspace.NewForwardGraphTransformer(graphProvider, dackbox.ImageToCVEPath.Path)),
			Options:  imageMappings.OptionsMap,
		},
	}...)
}

func wrapDerivedFieldSearcher(graphProvider graph.Provider, searcher search.Searcher) search.Searcher {
	if !features.Dackbox.Enabled() {
		return searcher
	}
	return derivedfields.CountSortedSearcher(searcher, map[string]counter.DerivedFieldCounter{
		search.DeploymentCount.String(): counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.CVEToDeploymentPath, deploymentSAC.GetSACFilter(graphProvider)),
		search.ImageCount.String():      counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.CVEToImagePath, imageSAC.GetSACFilter(graphProvider)),
		search.ComponentCount.String():  counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.CVEToComponentPath, componentSAC.GetSACFilter(graphProvider)),
	})
}
