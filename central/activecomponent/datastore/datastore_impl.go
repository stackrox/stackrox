package datastore

import (
	"context"

	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	"github.com/stackrox/rox/central/activecomponent/datastore/search"
	sacFilters "github.com/stackrox/rox/central/activecomponent/sac"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
)

type datastoreImpl struct {
	storage       store.Store
	graphProvider graph.Provider
	searcher      search.Searcher
}

func (ds *datastoreImpl) Search(ctx context.Context, query *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, query)
}

func (ds *datastoreImpl) SearchRawActiveComponents(ctx context.Context, query *v1.Query) ([]*storage.ActiveComponent, error) {
	return ds.searcher.SearchRawActiveComponents(ctx, query)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ActiveComponent, bool, error) {
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return nil, false, err
	}

	activeComponent, found, err := ds.storage.Get(id)
	if err != nil || !found {
		return nil, false, err
	}
	return activeComponent, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return false, err
	}

	found, err := ds.storage.Exists(id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ActiveComponent, error) {
	filteredIDs, err := ds.filterReadable(ctx, ids)
	if err != nil {
		return nil, err
	}

	activeComponents, _, err := ds.storage.GetBatch(filteredIDs)
	if err != nil {
		return nil, err
	}
	return activeComponents, nil
}

func (ds *datastoreImpl) filterReadable(ctx context.Context, ids []string) ([]string, error) {
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}
