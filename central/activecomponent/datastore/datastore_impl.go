package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/activecomponent/datastore/index"
	"github.com/stackrox/rox/central/activecomponent/datastore/internal/store"
	"github.com/stackrox/rox/central/activecomponent/datastore/search"
	sacFilters "github.com/stackrox/rox/central/activecomponent/sac"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
)

var (
	deploymentSAC = sac.ForResource(resources.Deployment)
)

type datastoreImpl struct {
	storage       store.Store
	graphProvider graph.Provider
	indexer       index.Indexer
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

	activeComponent, found, err := ds.storage.Get(ctx, id)
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

	found, err := ds.storage.Exists(ctx, id)
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

	activeComponents, _, err := ds.storage.GetMany(ctx, filteredIDs)
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

// UpsertBatch inserts active components
func (ds *datastoreImpl) UpsertBatch(ctx context.Context, acs []*storage.ActiveComponent) error {
	if ok, err := deploymentSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := ds.storage.UpsertMany(ctx, acs); err != nil {
		return errors.Wrap(err, "upserting active components to store")
	}
	return nil
}

// DeleteBatch deletes active components
func (ds *datastoreImpl) DeleteBatch(ctx context.Context, ids ...string) error {
	if ok, err := deploymentSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := ds.storage.DeleteMany(ctx, ids); err != nil {
		return errors.Wrap(err, "deleting active components")
	}
	return nil
}
