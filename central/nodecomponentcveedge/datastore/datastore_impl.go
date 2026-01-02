package datastore

import (
	"context"

	pgStore "github.com/stackrox/rox/central/nodecomponentcveedge/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage pgStore.Store
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.storage.Search(ctx, q)
}

func (ds *datastoreImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if q == nil {
		q = searchPkg.EmptyQuery()
	}

	results, err := ds.storage.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	// Populate Name from ID for each result
	for i := range results {
		results[i].Name = results[i].ID
	}

	return searchPkg.ResultsToSearchResultProtos(results, &NodeComponentCVEEdgeSearchResultConverter{}), nil
}

func (ds *datastoreImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.NodeComponentCVEEdge, error) {
	var edges []*storage.NodeComponentCVEEdge
	err := ds.storage.GetByQueryFn(ctx, q, func(edge *storage.NodeComponentCVEEdge) error {
		edges = append(edges, edge)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return edges, nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.NodeComponentCVEEdge, bool, error) {
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

type NodeComponentCVEEdgeSearchResultConverter struct{}

func (c *NodeComponentCVEEdgeSearchResultConverter) BuildName(result *searchPkg.Result) string {
	// Name is already populated from ID
	return result.Name
}

func (c *NodeComponentCVEEdgeSearchResultConverter) BuildLocation(result *searchPkg.Result) string {
	// NodeComponentCVEEdge does not have a location
	return ""
}

func (c *NodeComponentCVEEdgeSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_NODE_COMPONENT_CVE_EDGE
}

func (c *NodeComponentCVEEdgeSearchResultConverter) GetScore(result *searchPkg.Result) float64 {
	return result.Score
}
