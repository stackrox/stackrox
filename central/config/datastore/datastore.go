package datastore

import (
	"context"

	"github.com/stackrox/stackrox/central/config/store"
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/sac"
)

// DataStore is the entry point for modifying Config data.
//go:generate mockgen-wrapper
type DataStore interface {
	GetConfig(context.Context) (*storage.Config, error)
	UpsertConfig(context.Context, *storage.Config) error
}

// New returns an instance of DataStore.
func New(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

var (
	configSAC = sac.ForResource(resources.Config)
)

type datastoreImpl struct {
	store store.Store
}

// GetConfig returns Central's config
func (d *datastoreImpl) GetConfig(ctx context.Context) (*storage.Config, error) {
	if ok, err := configSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	conf, _, err := d.store.Get(ctx)
	return conf, err
}

// UpsertConfig updates Central's config
func (d *datastoreImpl) UpsertConfig(ctx context.Context, config *storage.Config) error {
	if ok, err := configSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.store.Upsert(ctx, config)
}
