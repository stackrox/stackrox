package search

import (
	"context"

	componentCVEEdgeMappings "github.com/stackrox/stackrox/central/componentcveedge/mappings"
	"github.com/stackrox/stackrox/central/cve/edgefields"
	cveMappings "github.com/stackrox/stackrox/central/cve/mappings"
	"github.com/stackrox/stackrox/central/dackbox"
	componentMappings "github.com/stackrox/stackrox/central/imagecomponent/mappings"
	"github.com/stackrox/stackrox/central/node/datastore/internal/store"
	"github.com/stackrox/stackrox/central/node/index"
	nodeMappings "github.com/stackrox/stackrox/central/node/index/mappings"
	nodeSAC "github.com/stackrox/stackrox/central/node/sac"
	nodeComponentEdgeMappings "github.com/stackrox/stackrox/central/nodecomponentedge/mappings"
	nodeComponentEdgeSAC "github.com/stackrox/stackrox/central/nodecomponentedge/sac"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
	"github.com/stackrox/stackrox/pkg/search/compound"
	"github.com/stackrox/stackrox/pkg/search/filtered"
	"github.com/stackrox/stackrox/pkg/search/paginated"
	"github.com/stackrox/stackrox/pkg/search/scoped"
	"github.com/stackrox/stackrox/pkg/search/sortfields"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.LastUpdatedTime.String(),
	}
	componentOptionsMap = search.CombineOptionsMaps(componentMappings.OptionsMap)
	nodeOnlyOptionsMap  = search.Difference(
		nodeMappings.OptionsMap,
		search.CombineOptionsMaps(
			nodeComponentEdgeMappings.OptionsMap,
			componentOptionsMap,
			componentCVEEdgeMappings.OptionsMap,
			cveMappings.OptionsMap,
		),
	)
)

// searcherImpl provides an intermediary implementation layer for node search.
type searcherImpl struct {
	storage       store.Store
	indexer       index.Indexer
	graphProvider graph.Provider
	searcher      search.Searcher
}

// SearchNodes retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchNodes(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	nodes, results, err := ds.searchNodes(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(nodes))
	for i, node := range nodes {
		protoResults = append(protoResults, convertNode(node, results[i]))
	}
	return protoResults, nil
}

// SearchRawNodes retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchRawNodes(ctx context.Context, q *v1.Query) ([]*storage.Node, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	nodes, _, err := ds.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func (ds *searcherImpl) searchNodes(ctx context.Context, q *v1.Query) ([]*storage.Node, []search.Result, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	nodes := make([]*storage.Node, 0, len(results))
	newResults := make([]search.Result, 0, len(results))
	for _, result := range results {
		node, exists, err := ds.storage.Get(ctx, result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		nodes = append(nodes, node)
		newResults = append(newResults, result)
	}
	return nodes, newResults, nil
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

// convertNode returns proto search result from a node object and the internal search result
func convertNode(node *storage.Node, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_NODES,
		Id:             node.GetId(),
		Name:           node.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(cveIndexer blevesearch.UnsafeSearcher,
	componentCVEEdgeIndexer blevesearch.UnsafeSearcher,
	componentIndexer blevesearch.UnsafeSearcher,
	nodeComponentEdgeIndexer blevesearch.UnsafeSearcher,
	nodeIndexer blevesearch.UnsafeSearcher) search.Searcher {

	cveSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(cveIndexer)
	componentCVEEdgeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentCVEEdgeIndexer)
	componentSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(componentIndexer)
	nodeComponentEdgeSearcher := filtered.UnsafeSearcher(nodeComponentEdgeIndexer, nodeComponentEdgeSAC.GetSACFilter())
	nodeSearcher := filtered.UnsafeSearcher(nodeIndexer, nodeSAC.GetSACFilter())

	compoundSearcher := getCompoundNodeSearcher(
		cveSearcher,
		componentCVEEdgeSearcher,
		componentSearcher,
		nodeComponentEdgeSearcher,
		nodeSearcher,
	)
	filteredSearcher := filtered.Searcher(edgefields.HandleCVEEdgeSearchQuery(compoundSearcher), nodeSAC.GetSACFilter())
	// To transform Component sort field to Component+Component Version.
	transformedSortSearcher := sortfields.TransformSortFields(filteredSearcher, nodeMappings.OptionsMap)
	paginatedSearcher := paginated.Paginated(transformedSortSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}

func getCompoundNodeSearcher(
	cveSearcher search.Searcher,
	componentCVEEdgeSearcher search.Searcher,
	componentSearcher search.Searcher,
	nodeComponentEdgeSearcher search.Searcher,
	nodeSearcher search.Searcher,
) search.Searcher {
	// The ordering of these is important, so do not change.
	return compound.NewSearcher([]compound.SearcherSpec{
		{
			Searcher:       scoped.WithScoping(cveSearcher, dackbox.ToCategory(v1.SearchCategory_VULNERABILITIES)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_NODES],
			Options:        cveMappings.OptionsMap,
		},
		{
			Searcher:       scoped.WithScoping(componentCVEEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_COMPONENT_VULN_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_COMPONENT_VULN_EDGE][v1.SearchCategory_NODES],
			Options:        componentCVEEdgeMappings.OptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_VULNERABILITIES][v1.SearchCategory_COMPONENT_VULN_EDGE],
		},
		{
			Searcher:       scoped.WithScoping(componentSearcher, dackbox.ToCategory(v1.SearchCategory_IMAGE_COMPONENTS)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENTS][v1.SearchCategory_NODES],
			Options:        componentOptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_COMPONENT_VULN_EDGE][v1.SearchCategory_IMAGE_COMPONENTS],
		},
		{
			Searcher:       scoped.WithScoping(nodeComponentEdgeSearcher, dackbox.ToCategory(v1.SearchCategory_NODE_COMPONENT_EDGE)),
			Transformation: dackbox.GraphTransformations[v1.SearchCategory_NODE_COMPONENT_EDGE][v1.SearchCategory_NODES],
			Options:        nodeComponentEdgeMappings.OptionsMap,
			LinkToPrev:     dackbox.GraphTransformations[v1.SearchCategory_IMAGE_COMPONENTS][v1.SearchCategory_NODE_COMPONENT_EDGE],
		},
		{
			IsDefault:  true,
			Searcher:   scoped.WithScoping(nodeSearcher, dackbox.ToCategory(v1.SearchCategory_NODES)),
			Options:    nodeOnlyOptionsMap,
			LinkToPrev: dackbox.GraphTransformations[v1.SearchCategory_NODE_COMPONENT_EDGE][v1.SearchCategory_NODES],
		},
	})
}
