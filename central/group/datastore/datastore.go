package datastore

import (
	"context"

	"github.com/stackrox/rox/central/group/datastore/internal/store"
	"github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
)

// DataStore is the datastore for groups.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Get(ctx context.Context, props *storage.GroupProperties) (*storage.Group, error)
	GetAll(ctx context.Context) ([]*storage.Group, error)
	GetFiltered(ctx context.Context, filter func(*storage.Group) bool) ([]*storage.Group, error)

	Walk(ctx context.Context, authProviderID string, attributes map[string][]string) ([]*storage.Group, error)

	Add(ctx context.Context, group *storage.Group) error
	Update(ctx context.Context, group *storage.Group, force bool) error
	Upsert(ctx context.Context, group *storage.Group) error
	Mutate(ctx context.Context, remove, update, add []*storage.Group, force bool) error
	Remove(ctx context.Context, props *storage.GroupProperties, force bool) error
	RemoveAllWithAuthProviderID(ctx context.Context, authProviderID string, force bool) error
	RemoveAllWithEmptyProperties(ctx context.Context) error
}

// New returns a new DataStore instance.
func New(storage store.Store, roleDatastore datastore.DataStore, authProviderDatastore authproviders.Store) DataStore {
	return &dataStoreImpl{
		storage:               storage,
		roleDatastore:         roleDatastore,
		authProviderDatastore: authProviderDatastore,
	}
}
