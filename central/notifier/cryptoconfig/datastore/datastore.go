package datastore

import (
	"context"

	pgstore "github.com/stackrox/rox/central/notifier/cryptoconfig/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	configCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.InstallationInfo)))
)

// DataStore is the entry point for modifying Config data.
//
//go:generate mockgen-wrapper
type DataStore interface {
	GetConfig() (*storage.NotifierCryptoConfig, error)
	UpsertConfig(*storage.NotifierCryptoConfig) error
}

// New returns an instance of DataStore.
func New(store pgstore.Store) DataStore {
	return &datastoreImpl{
		store: store,
	}
}

type datastoreImpl struct {
	store pgstore.Store
}

// GetConfig returns notifier crypto config
func (d *datastoreImpl) GetConfig() (*storage.NotifierCryptoConfig, error) {
	config, _, err := d.store.Get(configCtx)
	return config, err
}

// UpsertConfig upserts notifier crypto config
func (d *datastoreImpl) UpsertConfig(config *storage.NotifierCryptoConfig) error {
	return d.store.Upsert(configCtx, config)
}
