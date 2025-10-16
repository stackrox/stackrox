package datastore

import (
	"context"

	"github.com/pkg/errors"
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
	// TODO(ROX-29943): remove 2 pass database queries
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	edges, missingIndices, err := ds.storage.GetMany(ctx, searchPkg.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	results = searchPkg.RemoveMissingResults(results, missingIndices)
	return convertMany(edges, results)
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

func convertMany(edges []*storage.NodeComponentEdge, results []searchPkg.Result) ([]*v1.SearchResult, error) {
	if len(edges) != len(results) {
		return nil, errors.Errorf("expected %d edges but got %d", len(results), len(edges))
	}

	outputResults := make([]*v1.SearchResult, len(edges))
	for index, sar := range edges {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults, nil
}

func convertOne(obj *storage.NodeComponentEdge, result *searchPkg.Result) *v1.SearchResult {
	sr := &v1.SearchResult{}
	sr.SetCategory(v1.SearchCategory_NODE_COMPONENT_EDGE)
	sr.SetId(obj.GetId())
	sr.SetName(obj.GetId())
	sr.SetFieldToMatches(searchPkg.GetProtoMatchesMap(result.Matches))
	sr.SetScore(result.Score)
	return sr
}
