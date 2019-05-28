package datastore

import (
	"context"

	"github.com/stackrox/rox/central/role/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for roles.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	GetRole(ctx context.Context, name string) (*storage.Role, error)
	GetAllRoles(ctx context.Context) ([]*storage.Role, error)

	AddRole(ctx context.Context, role *storage.Role) error
	UpdateRole(ctx context.Context, role *storage.Role) error
	RemoveRole(ctx context.Context, name string) error
}

// New returns a new DataStore instance.
func New(storage store.Store) DataStore {
	return &dataStoreImpl{
		storage: storage,
	}
}
