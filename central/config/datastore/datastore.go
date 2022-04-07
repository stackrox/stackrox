package datastore

import (
	"context"

	"github.com/stackrox/rox/central/config/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
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

	return d.store.GetConfig()
}

// UpsertConfig updates Central's config
func (d *datastoreImpl) UpsertConfig(ctx context.Context, config *storage.Config) error {
	if ok, err := configSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.store.UpsertConfig(config)
}
