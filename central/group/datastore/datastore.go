package datastore

import (
	"context"

	"github.com/stackrox/rox/central/group/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for groups.
//go:generate mockgen-wrapper
type DataStore interface {
	Get(ctx context.Context, id string) (*storage.Group, error)
	GetAll(ctx context.Context) ([]*storage.Group, error)
	GetFiltered(ctx context.Context, filter func(*storage.GroupProperties) bool) ([]*storage.Group, error)

	Walk(ctx context.Context, authProviderID string, attributes map[string][]string) ([]*storage.Group, error)

	Add(ctx context.Context, group *storage.Group) error
	Update(ctx context.Context, group *storage.Group) error
	Upsert(ctx context.Context, group *storage.Group) error
	Mutate(ctx context.Context, remove, update, add []*storage.Group) error
	Remove(ctx context.Context, id string) error
	RemoveAllWithAuthProviderID(ctx context.Context, authProviderID string) error
}

// New returns a new DataStore instance.
func New(storage store.Store) DataStore {
	return &dataStoreImpl{
		storage: storage,
	}
}
