package datastore

import (
	"context"

	sacFilters "github.com/stackrox/rox/central/imagecveedge/sac"
	"github.com/stackrox/rox/central/imagecveedge/search"
	"github.com/stackrox/rox/central/imagecveedge/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/graph"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	pkgSearch "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/filtered"
)

var (
	log = logging.LoggerForModule()

	imagesSAC = sac.ForResource(resources.Image)
)

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
	filteredIDs, err := ds.filterReadable(ctx, []string{id})
	if err != nil || len(filteredIDs) != 1 {
		return nil, false, err
	}

	edge, found, err := ds.storage.Get(id)
	if err != nil || !found {
		return nil, false, err
	}
	return edge, true, nil
}

func (ds *datastoreImpl) UpdateVulnerabilityState(ctx context.Context, cve string, images []string, state storage.VulnerabilityState) error {
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if err := ds.storage.UpdateVulnState(cve, images, state); err != nil {
		return err
	}
	return nil
}

func (ds *datastoreImpl) filterReadable(ctx context.Context, ids []string) ([]string, error) {
	var filteredIDs []string
	var err error
	graph.Context(ctx, ds.graphProvider, func(graphContext context.Context) {
		filteredIDs, err = filtered.ApplySACFilter(graphContext, ids, sacFilters.GetSACFilter())
	})
	return filteredIDs, err
}
