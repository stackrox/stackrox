package datastore

import (
	"context"

	"github.com/stackrox/rox/central/sac"
	"github.com/stackrox/rox/central/sac/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"go.etcd.io/bbolt"
)

// DataStore exposes the functions that exposed and modify auth plugin configs in the system.
//go:generate mockgen-wrapper
type DataStore interface {
	ListAuthzPluginConfigs(ctx context.Context) ([]*storage.AuthzPluginConfig, error)
	GetAuthzPluginConfig(ctx context.Context, id string) (*storage.AuthzPluginConfig, error)
	UpsertAuthzPluginConfig(ctx context.Context, config *storage.AuthzPluginConfig) (*storage.AuthzPluginConfig, error)
	DeleteAuthzPluginConfig(ctx context.Context, id string) error
}

// New returns a new instance of a DataStore.
func New(db *bbolt.DB, clientMgr sac.AuthPluginClientManger) (DataStore, error) {
	storage, err := store.New(db)
	if err != nil {
		return nil, err
	}
	dataStore := &datastoreImpl{
		storage:   storage,
		clientMgr: clientMgr,
	}
	err = dataStore.Initialize()
	if err != nil {
		return nil, err
	}

	return dataStore, nil
}
