package datastore

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/imagecomponent/index"
	"github.com/stackrox/rox/central/imagecomponent/search"
	"github.com/stackrox/rox/central/imagecomponent/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

var (
	// TODO: Need to setup sac for Image Components correctly instead of relying on global access.
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

func (ds *datastoreImpl) SearchImageComponents(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}
	return ds.searcher.SearchImageComponents(ctx, q)
}

func (ds *datastoreImpl) SearchRawImageComponents(ctx context.Context, q *v1.Query) ([]*storage.ImageComponent, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}
	components, err := ds.searcher.SearchRawImageComponents(ctx, q)
	if err != nil {
		return nil, err
	}
	return components, nil
}

func (ds *datastoreImpl) Count(ctx context.Context) (int, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil || !ok {
		return 0, err
	}
	return ds.storage.Count()
}

func (ds *datastoreImpl) Get(ctx context.Context, id string) (*storage.ImageComponent, bool, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, false, err
	}
	component, found, err := ds.storage.Get(id)
	if err != nil || !found {
		return nil, false, err
	}
	return component, true, nil
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

func (ds *datastoreImpl) GetBatch(ctx context.Context, ids []string) ([]*storage.ImageComponent, error) {
	if ok, err := imagesSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}
	components, _, err := ds.storage.GetBatch(ids)
	if err != nil {
		return nil, err
	}
	return components, nil
}

// UpsertImage dedupes the image with the underlying storage and adds the image to the index.
func (ds *datastoreImpl) Upsert(ctx context.Context, imagecomponent *storage.ImageComponent) error {
	if imagecomponent.GetId() == "" {
		return errors.New("cannot upsert a component without an id")
	}
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	if err := ds.indexer.AddImageComponent(imagecomponent); err != nil {
		return err
	}
	return ds.storage.Upsert(imagecomponent)
}

func (ds *datastoreImpl) Delete(ctx context.Context, ids ...string) error {
	if ok, err := imagesSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	errorList := errorhelpers.NewErrorList("deleting components")
	for _, id := range ids {
		if err := ds.storage.Delete(id); err != nil {
			errorList.AddError(err)
			continue
		}
		if err := ds.indexer.DeleteImageComponent(id); err != nil {
			errorList.AddError(err)
		}
	}
	return errorList.ToError()
}
