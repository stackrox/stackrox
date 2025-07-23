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
	return ds.searcher.Count(ctx, searchPkg.EmptyQuery())
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

func convertMany(cves []*storage.NodeComponentEdge, results []searchPkg.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(cves))
	for idx, sar := range cves {
		outputResults[idx] = convertOne(sar, &results[idx])
	}
	return outputResults
}

func convertOne(cve *storage.NodeComponentEdge, result *searchPkg.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_NODE_COMPONENT_EDGE,
		Id:             cve.GetId(),
		Name:           cve.GetId(),
		FieldToMatches: searchPkg.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
