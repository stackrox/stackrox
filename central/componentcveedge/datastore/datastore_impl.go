package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/componentcveedge/index"
	"github.com/stackrox/rox/central/componentcveedge/search"
	"github.com/stackrox/rox/central/componentcveedge/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	imagesSAC = sac.ForResource(resources.Image)
)

type datastoreImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}
	return ds.searcher.Search(ctx, q)
}

func (ds *datastoreImpl) SearchEdges(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}
	return ds.searcher.SearchEdges(ctx, q)
}

func (ds *datastoreImpl) SearchRawEdges(ctx context.Context, q *v1.Query) ([]*storage.ComponentCVEEdge, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}
	imgs, err := ds.searcher.SearchRawEdges(ctx, q)
	if err != nil {
		return nil, err
	}
	return imgs, nil
}

func (ds *datastoreImpl) Count(ctx context.Context) (int, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); !ok || err != nil {
		return 0, err
	}
	return ds.storage.Count()
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ComponentCVEEdge, bool, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}
	edge, found, err := ds.storage.Get(id)
	if err != nil || !found {
		return nil, false, err
	}
	return edge, true, nil
}

func (ds *datastoreImpl) Exists(ctx context.Context, id string) (bool, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil || !ok {
		return false, err
	}
	found, err := ds.storage.Exists(id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
}

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ComponentCVEEdge, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}
	edges, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return nil, err
	}
	return edges, nil
}

func (ds *datastoreImpl) Upsert(ctx context.Context, edges ...*storage.ComponentCVEEdge) error {
	if len(edges) == 0 {
		return nil
	}
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := ds.storage.Upsert(edges...); err != nil {
		return err
	}
	return ds.indexer.AddComponentCVEEdges(edges)
}

func (ds *datastoreImpl) Delete(ctx context.Context, ids ...string) error {
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := ds.storage.Delete(ids...); err != nil {
		return err
	}
	return ds.indexer.DeleteComponentCVEEdges(ids)
}
