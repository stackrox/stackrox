package datastore

import (
	"context"

	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/watchedimage/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	watchedImageSAC = sac.ForResource(resources.WatchedImage)
)

type dataStore struct {
	storage store.Store
}

func (d *dataStore) UnwatchImage(ctx context.Context, name string) error {
	if ok, err := watchedImageSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}
	return d.storage.Delete(name)
}

func (d *dataStore) Exists(ctx context.Context, name string) (bool, error) {
	if ok, err := watchedImageSAC.ReadAllowed(ctx); err != nil || !ok {
		return false, err
	}
	return d.storage.Exists(name)
}

func (d *dataStore) UpsertWatchedImage(ctx context.Context, name string) error {
	if ok, err := watchedImageSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrPermissionDenied
	}
	return d.storage.Upsert(&storage.WatchedImage{Name: name})
}

func (d *dataStore) GetAllWatchedImages(ctx context.Context) ([]*storage.WatchedImage, error) {
	if ok, err := watchedImageSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	var watchedImages []*storage.WatchedImage
	err := d.storage.Walk(func(obj *storage.WatchedImage) error {
		watchedImages = append(watchedImages, obj)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return watchedImages, nil
}
