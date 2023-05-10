package datastore

import (
	"context"
	"io"

	"github.com/stackrox/rox/central/blob/datastore/store"
	"github.com/stackrox/rox/generated/storage"
)

// Datastore provides access to the blob store
type Datastore interface {
	Upsert(ctx context.Context, obj *storage.Blob, reader io.Reader) error
	Get(ctx context.Context, name string, writer io.Writer) (*storage.Blob, bool, error)
	Delete(ctx context.Context, name string) error
}

// NewDatastore creates a new Blob datastore
func NewDatastore(store store.Store) Datastore {
	return &datastoreImpl{
		store: store,
	}
}

type datastoreImpl struct {
	store store.Store
}

// Upsert adds a new blob to the database
func (d *datastoreImpl) Upsert(ctx context.Context, obj *storage.Blob, reader io.Reader) error {
	return d.store.Upsert(ctx, obj, reader)
}

// Get retrieves a blob from the database
func (d *datastoreImpl) Get(ctx context.Context, name string, writer io.Writer) (*storage.Blob, bool, error) {
	return d.store.Get(ctx, name, writer)
}

// Delete removes a blob store from database
func (d *datastoreImpl) Delete(ctx context.Context, name string) error {
	return d.store.Delete(ctx, name)
}
