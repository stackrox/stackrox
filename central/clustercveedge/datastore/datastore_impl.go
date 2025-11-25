package datastore

import (
	"context"

	"github.com/stackrox/rox/central/clustercveedge/store"
	"github.com/stackrox/rox/central/cve/edgefields"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

type datastoreImpl struct {
	storage store.Store
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.storage.Search(ctx, edgefields.TransformFixableFieldsQuery(q))
}

func (ds *datastoreImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.storage.Search(ctx, edgefields.TransformFixableFieldsQuery(q))
	if err != nil {
		return nil, err
	}

	// Populate Name from ID for each result
	for i := range results {
		results[i].Name = results[i].ID
	}

	return searchPkg.ResultsToSearchResultProtos(results, &ClusterCVEEdgeSearchResultConverter{}), nil
}

func (ds *datastoreImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ClusterCVEEdge, error) {
	q = edgefields.TransformFixableFieldsQuery(q)

	var edges []*storage.ClusterCVEEdge
	err := ds.storage.GetByQueryFn(ctx, q, func(edge *storage.ClusterCVEEdge) error {
		edges = append(edges, edge)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return edges, nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.storage.Count(ctx, edgefields.TransformFixableFieldsQuery(q))
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ClusterCVEEdge, bool, error) {
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

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ClusterCVEEdge, error) {
	edges, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

type ClusterCVEEdgeSearchResultConverter struct{}

func (c *ClusterCVEEdgeSearchResultConverter) BuildName(result *searchPkg.Result) string {
	// Name is already populated from ID
	return result.Name
}

func (c *ClusterCVEEdgeSearchResultConverter) BuildLocation(result *searchPkg.Result) string {
	// ClusterCVEEdge does not have a location
	return ""
}

func (c *ClusterCVEEdgeSearchResultConverter) GetCategory() v1.SearchCategory {
	return v1.SearchCategory_CLUSTER_VULN_EDGE
}

func (c *ClusterCVEEdgeSearchResultConverter) GetScore(result *searchPkg.Result) float64 {
	return result.Score
}
