package datastore

import (
	"context"

	"github.com/stackrox/rox/central/clustercveedge/store"
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
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	cves, missingIndices, err := ds.storage.GetMany(ctx, searchPkg.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	results = searchPkg.RemoveMissingResults(results, missingIndices)
	return convertMany(cves, results), nil
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
	return ds.searcher.Count(ctx, q)
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

func convertMany(cves []*storage.ClusterCVEEdge, results []searchPkg.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(obj *storage.ClusterCVEEdge, result *searchPkg.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_CLUSTER_VULN_EDGE,
		Id:             obj.GetId(),
		Name:           obj.GetId(),
		FieldToMatches: searchPkg.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
