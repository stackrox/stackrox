package datastore

import (
	"context"

	roleStore "github.com/stackrox/rox/central/role/datastore/internal/store"
	rocksDBStore "github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/generated/storage"
)

// DataStore is the datastore for roles.
//go:generate mockgen-wrapper
type DataStore interface {
	GetRole(ctx context.Context, name string) (*storage.Role, error)
	GetAllRoles(ctx context.Context) ([]*storage.Role, error)

	AddRole(ctx context.Context, role *storage.Role) error
	UpdateRole(ctx context.Context, role *storage.Role) error
	RemoveRole(ctx context.Context, name string) error

	GetAccessScope(ctx context.Context, id string) (*storage.SimpleAccessScope, bool, error)
	GetAllAccessScopes(ctx context.Context) ([]*storage.SimpleAccessScope, error)
	AddAccessScope(ctx context.Context, scope *storage.SimpleAccessScope) error
	UpdateAccessScope(ctx context.Context, scope *storage.SimpleAccessScope) error
	RemoveAccessScope(ctx context.Context, id string) error
}

// New returns a new DataStore instance.
func New(roleStorage roleStore.Store, accessScopeStore rocksDBStore.SimpleAccessScopeStore) DataStore {
	return &dataStoreImpl{
		roleStorage:        roleStorage,
		accessScopeStorage: accessScopeStore,
	}
}
