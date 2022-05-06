package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/componentcveedge/index"
	sacFilters "github.com/stackrox/rox/central/componentcveedge/sac"
	"github.com/stackrox/rox/central/componentcveedge/search"
	"github.com/stackrox/rox/central/componentcveedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/features"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/search/postgres"
)

type componentCVEEdgePks struct {
	componentID string
	cveID       string
}

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

func (ds *datastoreImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ComponentCVEEdge, error) {
	imgs, err := ds.searcher.SearchRawEdges(ctx, q)
	if err != nil {
		return nil, err
	}
	return imgs, nil
}

func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ComponentCVEEdge, bool, error) {
	if !features.PostgresDatastore.Enabled() {
		filteredIDs, err := ds.filterReadable(ctx, []string{id})
		if err != nil || len(filteredIDs) != 1 {
			return nil, false, err
		}
	}

	var pks componentCVEEdgePks
	var err error
	if features.PostgresDatastore.Enabled() {
		pks, err = getPKs(id)
		if err != nil {
			return nil, false, err
		}
	}
	// For dackbox, we do not need all the primary keys.

	edge, found, err := ds.storage.Get(ctx, id, pks.componentID, pks.cveID)
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

	var pks componentCVEEdgePks
	var err error
	if features.PostgresDatastore.Enabled() {
		pks, err = getPKs(id)
		if err != nil {
			return false, err
		}
	}
	// For dackbox, we do not need all the primary keys.

	found, err := ds.storage.Exists(ctx, id, pks.componentID, pks.cveID)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) filterReadable(ctx context.Context, ids []string) ([]string, error) {
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}

func getPKs(id string) (componentCVEEdgePks, error) {
	parts := postgres.IDToParts(id)
	if len(parts) != 5 {
		return componentCVEEdgePks{}, errors.Errorf("unexpected number of primary keys (%v) found for component-cve relation. Expected 5 parts", parts)
	}

	compName := parts[0]
	compVersion := parts[1]
	compOS := parts[2]
	cve := parts[3]
	cveOS := parts[4]
	return componentCVEEdgePks{
		componentID: postgres.IDFromPks([]string{compName, compVersion, compOS}),
		cveID:       postgres.IDFromPks([]string{cve, cveOS}),
	}, nil
}
