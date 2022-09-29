package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clustercveedge/index"
	sacFilters "github.com/stackrox/rox/central/clustercveedge/sac"
	"github.com/stackrox/rox/central/clustercveedge/search"
	"github.com/stackrox/rox/central/clustercveedge/store"
	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	clustersSAC = sac.ForResource(resources.Cluster)
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

func (ds *datastoreImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ClusterCVEEdge, error) {
	imgs, err := ds.searcher.SearchRawEdges(ctx, q)
	if err != nil {
		return nil, err
	}
	return imgs, nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ClusterCVEEdge, bool, error) {
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return nil, false, err
	}
	edge, found, err := ds.storage.Get(ctx, id)
	if err != nil || !found {
		return nil, false, err
	}
	return edge, true, nil
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

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ClusterCVEEdge, error) {
	filteredIDs, err := ds.filterReadable(ctx, ids)
	if err != nil {
		return nil, err
	}

	edges, _, err := ds.storage.GetMany(ctx, filteredIDs)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

func (ds *datastoreImpl) filterReadable(ctx context.Context, ids []string) ([]string, error) {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return ids, nil
	}
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}

func (ds *datastoreImpl) Upsert(ctx context.Context, parts ...converter.ClusterCVEParts) error {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return utils.Should(errors.New("Unexpected cluster-cve edge upsert when running on Postgres"))
	}
	if len(parts) == 0 {
		return nil
	}

	if ok, err := clustersSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	// Store the new CVE data.
	return ds.storage.Upsert(ctx, parts...)
}

func (ds *datastoreImpl) Delete(ctx context.Context, ids ...string) error {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return utils.Should(errors.New("Unexpected cluster-cve edge upsert when running on Postgres"))
	}
	if ok, err := clustersSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.Delete(ctx, ids...)
}
