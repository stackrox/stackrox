package datastore

import (
	"context"
	"io"

	"github.com/stackrox/rox/central/blob/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// Datastore provides access to the blob store
type Datastore interface {
	GetNames(ctx context.Context) ([]string, error)
	Upsert(ctx context.Context, obj *storage.Blob, reader io.Reader) error
	Get(ctx context.Context, name string, writer io.Writer) (*storage.Blob, bool, error)
	Delete(ctx context.Context, name string) error
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
}

// NewDatastore creates a new Blob datastore
func NewDatastore(store store.Store, searcher search.Searcher) Datastore {
	return &datastoreImpl{
		store:    store,
		searcher: searcher,
	}
}

type datastoreImpl struct {
	store    store.Store
	searcher search.Searcher
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

// Search blobs
func (d *datastoreImpl) Search(ctx context.Context, query *v1.Query) ([]search.Result, error) {
	return d.searcher.Search(ctx, query)
}

// GetNames return all blob names
func (d *datastoreImpl) GetNames(ctx context.Context) ([]string, error) {
	return d.store.GetNames(ctx)
}
