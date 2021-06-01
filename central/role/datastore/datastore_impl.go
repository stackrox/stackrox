package datastore

import (
	"context"

	"github.com/pkg/errors"
	rolePkg "github.com/stackrox/rox/central/role"
	roleStore "github.com/stackrox/rox/central/role/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	rocksDBStore "github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/central/role/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	roleSAC = sac.ForResource(resources.Role)
)

type dataStoreImpl struct {
	roleStorage          roleStore.Store
	permissionSetStorage rocksDBStore.PermissionSetStore
	accessScopeStorage   rocksDBStore.SimpleAccessScopeStore

	lock sync.Mutex
}

func (ds *dataStoreImpl) GetRole(ctx context.Context, name string) (*storage.Role, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.roleStorage.GetRole(name)
}

func (ds *dataStoreImpl) GetAllRoles(ctx context.Context) ([]*storage.Role, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return ds.roleStorage.GetAllRoles()
}

func (ds *dataStoreImpl) AddRole(ctx context.Context, role *storage.Role) error {
	if ok, err := roleSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}
	if isDefaultRole(role) {
		return errors.Errorf("cannot modify default role %s", role.GetName())
	}

	return ds.roleStorage.AddRole(role)
}

func (ds *dataStoreImpl) UpdateRole(ctx context.Context, role *storage.Role) error {
	if ok, err := roleSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}
	if isDefaultRole(role) {
		return errors.Errorf("cannot modify default role %s", role.GetName())
	}

	return ds.roleStorage.UpdateRole(role)
}

func (ds *dataStoreImpl) RemoveRole(ctx context.Context, name string) error {
	if ok, err := roleSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}
	if isDefaultRoleName(name) {
		return errors.Errorf("cannot modify default role %s", name)
	}

	return ds.roleStorage.RemoveRole(name)
}

////////////////////////////////////////////////////////////////////////////////
// Permission sets                                                            //
//                                                                            //

func (ds *dataStoreImpl) GetPermissionSet(ctx context.Context, id string) (*storage.PermissionSet, bool, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}

	return ds.permissionSetStorage.Get(id)
}

func (ds *dataStoreImpl) GetAllPermissionSets(ctx context.Context) ([]*storage.PermissionSet, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var permissionSets []*storage.PermissionSet
	err := ds.permissionSetStorage.Walk(func(permissionSet *storage.PermissionSet) error {
		permissionSets = append(permissionSets, permissionSet)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return permissionSets, nil
}

func (ds *dataStoreImpl) AddPermissionSet(ctx context.Context, permissionSet *storage.PermissionSet) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := utils.ValidatePermissionSet(permissionSet); err != nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, err.Error())
	}
	if isDefaultRoleName(permissionSet.Name) {
		return errors.Errorf("cannot modify default role permission set %s", permissionSet.Name)
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyPermissionSetIDDoesNotExist(permissionSet.GetId()); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no permission set with the same name.
	if err := ds.permissionSetStorage.Upsert(permissionSet); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) UpdatePermissionSet(ctx context.Context, permissionSet *storage.PermissionSet) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := utils.ValidatePermissionSet(permissionSet); err != nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, err.Error())
	}
	if isDefaultRoleName(permissionSet.Name) {
		return errors.Errorf("cannot modify default role permission set %s", permissionSet.Name)
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyPermissionSetIDExists(permissionSet.GetId()); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no permission set with the same name.
	if err := ds.permissionSetStorage.Upsert(permissionSet); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) RemovePermissionSet(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	permissionSet, found, err := ds.permissionSetStorage.Get(id)
	if err != nil {
		return err
	}
	if !found {
		return errors.Wrapf(errorhelpers.ErrNotFound, "id = %q", id)
	}
	if isDefaultRoleName(permissionSet.Name) {
		return errors.Errorf("cannot modify default role permission set %s", permissionSet.Name)
	}

	// Constraints ok, delete the object.
	if err := ds.permissionSetStorage.Delete(id); err != nil {
		return err
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Access scopes                                                              //
//                                                                            //

func (ds *dataStoreImpl) GetAccessScope(ctx context.Context, id string) (*storage.SimpleAccessScope, bool, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}

	return ds.accessScopeStorage.Get(id)
}

func (ds *dataStoreImpl) GetAllAccessScopes(ctx context.Context) ([]*storage.SimpleAccessScope, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var scopes []*storage.SimpleAccessScope
	err := ds.accessScopeStorage.Walk(func(scope *storage.SimpleAccessScope) error {
		scopes = append(scopes, scope)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return scopes, nil
}

func (ds *dataStoreImpl) AddAccessScope(ctx context.Context, scope *storage.SimpleAccessScope) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := utils.ValidateSimpleAccessScope(scope); err != nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, err.Error())
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyAccessScopeIDDoesNotExist(scope.GetId()); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no access scope with the same name.
	if err := ds.accessScopeStorage.Upsert(scope); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) UpdateAccessScope(ctx context.Context, scope *storage.SimpleAccessScope) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := utils.ValidateSimpleAccessScope(scope); err != nil {
		return errors.Wrap(errorhelpers.ErrInvalidArgs, err.Error())
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyAccessScopeIDExists(scope.GetId()); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no access scope with the same name.
	if err := ds.accessScopeStorage.Upsert(scope); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) RemoveAccessScope(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyAccessScopeIDExists(id); err != nil {
		return err
	}

	// Constraints ok, delete the object.
	if err := ds.accessScopeStorage.Delete(id); err != nil {
		return err
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Storage constraints                                                        //
//                                                                            //
// Uniqueness of the 'name' field is expected to be verified by the           //
// underlying store, see its `--uniq-key-func` flag                           //

// Returns errorhelpers.ErrNotFound if there is no permission set with the supplied ID.
func (ds *dataStoreImpl) verifyPermissionSetIDExists(id string) error {
	_, found, err := ds.permissionSetStorage.Get(id)

	if err != nil {
		return err
	}
	if !found {
		return errors.Wrapf(errorhelpers.ErrNotFound, "id = %q", id)
	}
	return nil
}

// Returns errorhelpers.ErrAlreadyExists if there is a permission set with the same ID.
func (ds *dataStoreImpl) verifyPermissionSetIDDoesNotExist(id string) error {
	_, found, err := ds.permissionSetStorage.Get(id)

	if err != nil {
		return err
	}
	if found {
		return errors.Wrapf(errorhelpers.ErrAlreadyExists, "id = %q", id)
	}
	return nil
}

// Returns errorhelpers.ErrNotFound if there is no access scope with the supplied ID.
func (ds *dataStoreImpl) verifyAccessScopeIDExists(id string) error {
	_, found, err := ds.accessScopeStorage.Get(id)

	if err != nil {
		return err
	}
	if !found {
		return errors.Wrapf(errorhelpers.ErrNotFound, "id = %q", id)
	}
	return nil
}

// Returns errorhelpers.ErrAlreadyExists if there is an access scope with the same ID.
func (ds *dataStoreImpl) verifyAccessScopeIDDoesNotExist(id string) error {
	_, found, err := ds.accessScopeStorage.Get(id)

	if err != nil {
		return err
	}
	if found {
		return errors.Wrapf(errorhelpers.ErrAlreadyExists, "id = %q", id)
	}
	return nil
}

// Helper functions to check if a given role/name corresponds to a pre-loaded role.
func isDefaultRoleName(name string) bool {
	return rolePkg.DefaultRoleNames.Contains(name)
}

func isDefaultRole(role *storage.Role) bool {
	return isDefaultRoleName(role.GetName())
}
