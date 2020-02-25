package search

import (
	"context"
	"fmt"

	componentCVEEdgeMappings "github.com/stackrox/rox/central/componentcveedge/mappings"
	cveMappings "github.com/stackrox/rox/central/cve/mappings"
	cveSAC "github.com/stackrox/rox/central/cve/sac"
	"github.com/stackrox/rox/central/dackbox"
	"github.com/stackrox/rox/central/deployment/index"
	deploymentSAC "github.com/stackrox/rox/central/deployment/sac"
	"github.com/stackrox/rox/central/deployment/store"
	imageSAC "github.com/stackrox/rox/central/image/sac"
	componentMappings "github.com/stackrox/rox/central/imagecomponent/mappings"
	imageComponentEdgeMappings "github.com/stackrox/rox/central/imagecomponentedge/mappings"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/derivedfields/counter"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/compound"
	"github.com/stackrox/rox/pkg/search/derivedfields"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/search/idspace"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	imageMappings "github.com/stackrox/rox/pkg/search/options/images"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	deploymentsSearchHelper = sac.ForResource(resources.Deployment).MustCreateSearchHelper(deployments.OptionsMap)

	defaultSortOption = &v1.QuerySortOption{
		Field:    search.Priority.String(),
		Reversed: false,
	}
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
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
	deployments, missingIndices, err := ds.storage.ListDeploymentsWithIDs(ids...)
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
	deployments, _, err := ds.storage.GetDeploymentsWithIDs(ids...)
	if err != nil {
		return nil, err
	}
	return deployments, nil
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// ConvertDeployment returns proto search result from a deployment object and the internal search result
func convertDeployment(deployment *storage.ListDeployment, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_DEPLOYMENTS,
		Id:             deployment.GetId(),
		Name:           deployment.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
		Location:       fmt.Sprintf("/%s/%s", deployment.GetCluster(), deployment.GetNamespace()),
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

	var filteredSearcher search.Searcher
	if features.Dackbox.Enabled() {
		compoundSearcher := getDeploymentCompoundSearcher(graphProvider,
			cveSearcher,
			componentCVEEdgeSearcher,
			componentSearcher,
			imageComponentEdgeSearcher,
			imageSearcher,
			deploymentSearcher)
		filteredSearcher = filtered.Searcher(compoundSearcher, deploymentSAC.GetSACFilter(graphProvider)) // Make the UnsafeSearcher safe.
	} else {
		filteredSearcher = deploymentsSearchHelper.FilteredSearcher(deploymentIndexer) // Make the UnsafeSearcher safe.
	}

	transformedSortFieldSearcher := sortfields.TransformSortFields(filteredSearcher)
	derivedFieldSortedSearcher := wrapDerivedFieldSearcher(graphProvider, transformedSortFieldSearcher)
	paginatedSearcher := paginated.Paginated(derivedFieldSortedSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}

func getDeploymentCompoundSearcher(graphProvider idspace.GraphProvider,
	cveSearcher search.Searcher,
	componentCVEEdgeSearcher search.Searcher,
	componentSearcher search.Searcher,
	imageComponentEdgeSearcher search.Searcher,
	imageSearcher search.Searcher,
	deploymentSearcher search.Searcher) search.Searcher {
	cveEdgeToComponentSearcher := idspace.TransformIDs(componentCVEEdgeSearcher, idspace.NewEdgeToParentTransformer())
	imageComponentEdgeToImageSearcher := idspace.TransformIDs(imageComponentEdgeSearcher, idspace.NewEdgeToParentTransformer())
	return compound.NewSearcher([]compound.SearcherSpec{
		{
			Searcher: idspace.TransformIDs(cveSearcher, idspace.NewBackwardGraphTransformer(graphProvider, dackbox.CVEToDeploymentPath.Path)),
			Options:  cveMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(cveEdgeToComponentSearcher, idspace.NewBackwardGraphTransformer(graphProvider, dackbox.ComponentToDeploymentPath.Path)),
			Options:  componentCVEEdgeMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(componentSearcher, idspace.NewBackwardGraphTransformer(graphProvider, dackbox.ComponentToDeploymentPath.Path)),
			Options:  componentMappings.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(imageComponentEdgeToImageSearcher, idspace.NewBackwardGraphTransformer(graphProvider, dackbox.ImageToDeploymentPath.Path)),
			Options:  imageComponentEdgeMappings.OptionsMap,
		},
		{
			IsDefault: true,
			Searcher:  deploymentSearcher,
			Options:   deployments.OptionsMap,
		},
		{
			Searcher: idspace.TransformIDs(imageSearcher, idspace.NewBackwardGraphTransformer(graphProvider, dackbox.ImageToDeploymentPath.Path)),
			Options:  imageMappings.OptionsMap,
		},
	}...)
}

func wrapDerivedFieldSearcher(graphProvider graph.Provider, searcher search.Searcher) search.Searcher {
	if !features.Dackbox.Enabled() {
		return searcher
	}
	return derivedfields.CountSortedSearcher(searcher, map[string]counter.DerivedFieldCounter{
		search.ImageCount.String(): counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.DeploymentToImage, imageSAC.GetSACFilter(graphProvider)),
		search.CVECount.String():   counter.NewGraphBasedDerivedFieldCounter(graphProvider, dackbox.DeploymentToCVE, cveSAC.GetSACFilters(graphProvider)...),
	})
}
