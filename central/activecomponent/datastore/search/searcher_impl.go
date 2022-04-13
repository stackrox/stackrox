package search

import (
	"context"

	"github.com/stackrox/stackrox/central/activecomponent/datastore/internal/store"
	activeComponentMappings "github.com/stackrox/stackrox/central/activecomponent/index/mappings"
	activeComponentSAC "github.com/stackrox/stackrox/central/activecomponent/sac"
	cveMappings "github.com/stackrox/stackrox/central/cve/mappings"
	"github.com/stackrox/stackrox/central/dackbox"
	componentMappings "github.com/stackrox/stackrox/central/imagecomponent/mappings"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
	"github.com/stackrox/stackrox/pkg/search/compound"
	"github.com/stackrox/stackrox/pkg/search/filtered"
	deploymentMappings "github.com/stackrox/stackrox/pkg/search/options/deployments"
	"github.com/stackrox/stackrox/pkg/search/paginated"
	"github.com/stackrox/stackrox/pkg/search/scoped"
)

var (
	componentOptionsMap = search.CombineOptionsMaps(componentMappings.OptionsMap)
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage       store.Store
	graphProvider graph.Provider
	searcher      search.Searcher
}

// SearchRawActiveComponents retrieves activeComponents from the indexer and storage
func (ds *searcherImpl) SearchRawActiveComponents(ctx context.Context, q *v1.Query) ([]*storage.ActiveComponent, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	activeComponents, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return nil, err
	}
	return activeComponents, nil
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
func formatSearcher(acIndexer blevesearch.UnsafeSearcher, cveIndexer blevesearch.UnsafeSearcher, componentIndexer blevesearch.UnsafeSearcher, deploymentIndexer blevesearch.UnsafeSearcher) search.Searcher {
	acSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(acIndexer)
	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	componentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentIndexer)
	deploymentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(deploymentIndexer)

	compoundSearcher := getCompoundSearcher(acSearcher, cveSearcher, componentSearcher, deploymentSearcher)
	filteredSearcher := filtered.Searcher(compoundSearcher, activeComponentSAC.GetSACFilter())
	paginatedSearcher := paginated.Paginated(filteredSearcher)
	return paginatedSearcher
}

func getCompoundSearcher(acSearcher, cveSearcher, componentSearcher, deploymentSearcher search.Searcher) search.Searcher {
	// The ordering of these is important, so do not change.
	return compound.NewSearcher([]compound.SearcherSpec{
		{
			Searcher:       scoped.WithScoping(cveSearcher, dackbox.ToCategory(v1.SearchCategory_VULNERABILITIES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_ACTIVE_COMPONENT],
			Options:        cveMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(componentSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENTS][v1.SearchCategory_ACTIVE_COMPONENT],
			Options:        componentOptionsMap,
		},
		{
			IsDefault: true,
			Searcher:  scoped.WithScoping(acSearcher, dackbox.ToCategory(v1.SearchCategory_ACTIVE_COMPONENT)),
			Options:   activeComponentMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(deploymentSearcher, dackbox.ToCategory(v1.SearchCategory_DEPLOYMENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_DEPLOYMENTS][v1.SearchCategory_ACTIVE_COMPONENT],
			Options:        deploymentMappings.OptionsMap,
		},
	})
}
