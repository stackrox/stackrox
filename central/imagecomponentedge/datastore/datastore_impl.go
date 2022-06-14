package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/imagecomponentedge/index"
	sacFilters "github.com/stackrox/stackrox/central/imagecomponentedge/sac"
	"github.com/stackrox/stackrox/central/imagecomponentedge/search"
	"github.com/stackrox/stackrox/central/imagecomponentedge/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/features"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/filtered"
)

type datastoreImpl struct {
	storage       store.Store
	indexer       index.Indexer
	searcher      search.Searcher
	graphProvider graph.Provider
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchEdges(ctx, q)
}

func (ds *datastoreImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageComponentEdge, error) {
	edges, err := ds.searcher.SearchRawEdges(ctx, q)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

func (ds *datastoreImpl) Count(ctx context.Context) (int, error) {
	return ds.searcher.Count(ctx, searchPkg.EmptyQuery())
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageComponentEdge, bool, error) {
	if !features.PostgresDatastore.Enabled() {
		filteredIDs, err := ds.filterReadable(ctx, []string{id})
		if err != nil || len(filteredIDs) != 1 {
			return nil, false, err
		}
	}

	edge, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return edge, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	if !features.PostgresDatastore.Enabled() {
		filteredIDs, err := ds.filterReadable(ctx, []string{id})
		if err != nil || len(filteredIDs) != 1 {
			return false, err
		}
	}

	found, err := ds.storage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ImageComponentEdge, error) {
	if !features.PostgresDatastore.Enabled() {
		var err error
		ids, err = ds.filterReadable(ctx, ids)
		if err != nil {
			return nil, err
		}
	}

	edges, _, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

func (ds *datastoreImpl) filterReadable(ctx context.Context, ids []string) ([]string, error) {
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}
