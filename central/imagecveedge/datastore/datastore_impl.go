package datastore

import (
	"context"

	"github.com/pkg/errors"
	sacFilters "github.com/stackrox/rox/central/imagecveedge/sac"
	"github.com/stackrox/rox/central/imagecveedge/search"
	"github.com/stackrox/rox/central/imagecveedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/features"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
	"github.com/stackrox/rox/pkg/search/postgres"
)

type imageCVEEdgePks struct {
	imageID string
	cveID   string
}

type datastoreImpl struct {
	graphProvider graph.Provider
	storage       store.Store
	searcher      search.Searcher
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]pkgSearch.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchEdges(ctx, q)
}

func (ds *datastoreImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ImageCVEEdge, error) {
	edges, err := ds.searcher.SearchRawEdges(ctx, q)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageCVEEdge, bool, error) {
	if !features.PostgresDatastore.Enabled() {
		filteredIDs, err := ds.filterReadable(ctx, []string{id})
		if err != nil || len(filteredIDs) != 1 {
			return nil, false, err
		}
	}

	var pks imageCVEEdgePks
	var err error
	if features.PostgresDatastore.Enabled() {
		pks, err = getPKs(id)
		if err != nil {
			return nil, false, err
		}
	}
	// For dackbox, we do not need all the primary keys.

	edge, found, err := ds.storage.Get(ctx, id, pks.imageID, pks.cveID)
	if err != nil || !found {
		return nil, false, err
	}
	return edge, true, nil
}

func (ds *datastoreImpl) filterReadable(ctx context.Context, ids []string) ([]string, error) {
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}

func getPKs(id string) (imageCVEEdgePks, error) {
	parts := postgres.IDToParts(id)
	if len(parts) != 3 {
		return imageCVEEdgePks{}, errors.Errorf("unexpected number of primary keys (%v) found for component-cve relation. Expected 3 parts", parts)
	}

	imageID := parts[0]
	cve := parts[1]
	cveOS := parts[2]
	return imageCVEEdgePks{
		imageID: imageID,
		cveID:   postgres.IDFromPks([]string{cve, cveOS}),
	}, nil
}
