package datastore

import (
	"context"

	"github.com/stackrox/rox/central/hash/datastore/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	pg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/schema"
)

// Datastore implements the datastore interface for Hash objects
type Datastore interface {
	UpsertHash(ctx context.Context, hash *storage.Hash) error
	GetHashes(ctx context.Context, clusterID string) (*storage.Hash, bool, error)
	DeleteHashes(ctx context.Context, clusterID string) error
	TruncateHashes(ctx context.Context) error
}

// NewDatastore returns a new hash flush datastore
func NewDatastore(store postgres.Store, db pg.DB) Datastore {
	return &datastoreImpl{
		store: store,
		db:    db,
	}
}

type datastoreImpl struct {
	store postgres.Store
	db    pg.DB
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

// TruncateHashes truncates the hash table
func (d *datastoreImpl) TruncateHashes(ctx context.Context) error {
	_, err := d.db.Exec(ctx, "TRUNCATE TABLE "+schema.HashesTableName)
	return err
}
