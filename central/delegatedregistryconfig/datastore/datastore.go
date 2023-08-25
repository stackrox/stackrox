package datastore

import (
	"context"

	"github.com/stackrox/rox/central/delegatedregistryconfig/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

// DataStore is the entry point for modifying Delegated Registry Config data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetConfig(context.Context) (*storage.DelegatedRegistryConfig, bool, error)
	UpsertConfig(context.Context, *storage.DelegatedRegistryConfig) error
}

// New returns an instance of DataStore.
func New(store store.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

var (
	administrationSAC = sac.ForResource(resources.Administration)
)

type datastoreImpl struct {
	store store.Store
}

// GetConfig returns Central's delegated registry config
func (d *datastoreImpl) GetConfig(ctx context.Context) (*storage.DelegatedRegistryConfig, bool, error) {
	if ok, err := administrationSAC.ReadAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, nil
	}

	return d.store.Get(ctx)
}

// UpsertConfig updates Central's delegated registry config
func (d *datastoreImpl) UpsertConfig(ctx context.Context, config *storage.DelegatedRegistryConfig) error {
	if ok, err := administrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return d.store.Upsert(ctx, config)
}
