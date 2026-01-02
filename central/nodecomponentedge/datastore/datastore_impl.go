package datastore

import (
	"context"

	"github.com/stackrox/rox/central/nodecomponentedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage store.Store
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.storage.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	// Populate Name from ID for each result
	for i := range results {
		results[i].Name = results[i].ID
	}

	return searchPkg.ResultsToSearchResultProtos(results, &NodeComponentEdgeSearchResultConverter{}), nil
}

func (ds *datastoreImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentEdge, error) {
	var edges []*storage.NodeComponentEdge
	err := ds.storage.GetByQueryFn(ctx, q, func(edge *storage.NodeComponentEdge) error {
		edges = append(edges, edge)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return edges, nil
}

func (ds *datastoreImpl) Count(ctx context.Context) (int, error) {
	return ds.storage.Count(ctx, searchPkg.EmptyQuery())
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.NodeComponentEdge, bool, error) {
	edge, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return edge, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.NodeComponentEdge, error) {
	filteredIDs := ids
	edges, _, err := ds.storage.GetMany(ctx, filteredIDs)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

type NodeComponentEdgeSearchResultConverter struct{}

func (c *NodeComponentEdgeSearchResultConverter) BuildName(result *searchPkg.Result) string {
	// Name is already populated from ID
	return result.Name
}

func (c *NodeComponentEdgeSearchResultConverter) BuildLocation(result *searchPkg.Result) string {
	// NodeComponentEdge does not have a location
	return ""
}

func (c *NodeComponentEdgeSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_NODE_COMPONENT_EDGE
}

func (c *NodeComponentEdgeSearchResultConverter) GetScore(result *searchPkg.Result) float64 {
	return result.Score
}
