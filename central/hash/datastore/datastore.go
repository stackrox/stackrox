package datastore

import (
	"context"

	"github.com/stackrox/rox/central/hash/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
)

// Datastore implements the datastore interface for Hash objects
type Datastore interface {
	UpsertHash(ctx context.Context, hash *storage.Hash) error
	GetHashes(ctx context.Context, clusterID string) (*storage.Hash, bool, error)
	DeleteHashes(ctx context.Context, clusterID string) error
}

// NewDatastore returns a new hash flush datastore
func NewDatastore(store postgres.Store) Datastore {
	return &datastoreImpl{
		store: store,
	}
}

type datastoreImpl struct {
	store postgres.Store
}

// UpsertHash inserts the hash into the database
func (d *datastoreImpl) UpsertHash(ctx context.Context, hash *storage.Hash) error {
	return d.store.Upsert(ctx, hash)
}

// GetHashes gets the hashes for a particular cluster
func (d *datastoreImpl) GetHashes(ctx context.Context, clusterID string) (*storage.Hash, bool, error) {
	return d.store.Get(ctx, clusterID)
}

// DeleteHashes removes hashes from the database
func (d *datastoreImpl) DeleteHashes(ctx context.Context, clusterID string) error {
	return d.store.Delete(ctx, clusterID)
}
