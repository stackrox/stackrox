package datastore

import (
	"context"

	rocksDBStore "github.com/stackrox/stackrox/central/role/store"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
)

// DataStore is the datastore for roles.
//go:generate mockgen-wrapper
type DataStore interface {
	GetRole(ctx context.Context, name string) (*storage.Role, bool, error)
	GetAllRoles(ctx context.Context) ([]*storage.Role, error)
	AddRole(ctx context.Context, role *storage.Role) error
	UpdateRole(ctx context.Context, role *storage.Role) error
	RemoveRole(ctx context.Context, name string) error

	GetPermissionSet(ctx context.Context, id string) (*storage.PermissionSet, bool, error)
	GetAllPermissionSets(ctx context.Context) ([]*storage.PermissionSet, error)
	AddPermissionSet(ctx context.Context, permissionSet *storage.PermissionSet) error
	UpdatePermissionSet(ctx context.Context, permissionSet *storage.PermissionSet) error
	RemovePermissionSet(ctx context.Context, id string) error

	GetAccessScope(ctx context.Context, id string) (*storage.SimpleAccessScope, bool, error)
	GetAllAccessScopes(ctx context.Context) ([]*storage.SimpleAccessScope, error)
	AddAccessScope(ctx context.Context, scope *storage.SimpleAccessScope) error
	UpdateAccessScope(ctx context.Context, scope *storage.SimpleAccessScope) error
	RemoveAccessScope(ctx context.Context, id string) error

	GetAndResolveRole(ctx context.Context, name string) (permissions.ResolvedRole, error)
}

// New returns a new DataStore instance.
func New(roleStorage rocksDBStore.RoleStore, permissionSetStore rocksDBStore.PermissionSetStore, accessScopeStore rocksDBStore.SimpleAccessScopeStore) DataStore {
	return &dataStoreImpl{
		roleStorage:          roleStorage,
		permissionSetStorage: permissionSetStore,
		accessScopeStorage:   accessScopeStore,
	}
}
