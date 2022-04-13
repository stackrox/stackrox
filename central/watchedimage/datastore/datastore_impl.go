package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/central/watchedimage/datastore/internal/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
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
		return sac.ErrResourceAccessDenied
	}
	return d.storage.Delete(ctx, name)
}

func (d *dataStore) Exists(ctx context.Context, name string) (bool, error) {
	if ok, err := watchedImageSAC.ReadAllowed(ctx); err != nil || !ok {
		return false, err
	}
	return d.storage.Exists(ctx, name)
}

func (d *dataStore) UpsertWatchedImage(ctx context.Context, name string) error {
	if ok, err := watchedImageSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	return d.storage.Upsert(ctx, &storage.WatchedImage{Name: name})
}

func (d *dataStore) GetAllWatchedImages(ctx context.Context) ([]*storage.WatchedImage, error) {
	if ok, err := watchedImageSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	var watchedImages []*storage.WatchedImage
	err := d.storage.Walk(ctx, func(obj *storage.WatchedImage) error {
		watchedImages = append(watchedImages, obj)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return watchedImages, nil
}
