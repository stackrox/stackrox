package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/imagecomponentedge/index"
	sacFilters "github.com/stackrox/rox/central/imagecomponentedge/sac"
	"github.com/stackrox/rox/central/imagecomponentedge/search"
	"github.com/stackrox/rox/central/imagecomponentedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/features"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/search/postgres"
)

type imageComponentEdgePks struct {
	imageID     string
	compID      string
	compName    string
	compVersion string
	compOS      string
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
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return nil, false, err
	}

	var pks imageComponentEdgePks
	if features.PostgresDatastore.Enabled() {
		pks, err = getPKs(id)
		if err != nil {
			return nil, false, err
		}
	}
	// For dackbox, we do not need all the primary keys.

	edge, found, err := ds.storage.Get(ctx, id, pks.imageID, pks.compID, pks.compName, pks.compVersion, pks.compOS)
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

	var pks imageComponentEdgePks
	if features.PostgresDatastore.Enabled() {
		pks, err = getPKs(id)
		if err != nil {
			return false, err
		}
	}
	// For dackbox, we do not need all the primary keys.

	found, err := ds.storage.Exists(ctx, id, pks.imageID, pks.compID, pks.compName, pks.compVersion, pks.compOS)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ImageComponentEdge, error) {
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
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}

func getPKs(id string) (imageComponentEdgePks, error) {
	parts := postgres.IDToParts(id)
	if len(parts) != 4 {
		return imageComponentEdgePks{}, errors.Errorf("unexpected number of primary keys (%v) found for image-component relation. Expected 4 parts", parts)
	}

	return imageComponentEdgePks{
		imageID:     parts[0],
		compID:      parts[1],
		compName:    parts[2],
		compVersion: parts[3],
		compOS:      parts[4],
	}, nil
}
