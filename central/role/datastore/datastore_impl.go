package datastore

import (
	"context"

	"github.com/pkg/errors"
	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/central/role/resources"
	rocksDBStore "github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	roleSAC = sac.ForResource(resources.Role)

	log = logging.LoggerForModule()
)

type dataStoreImpl struct {
	roleStorage          rocksDBStore.RoleStore
	permissionSetStorage rocksDBStore.PermissionSetStore
	accessScopeStorage   rocksDBStore.SimpleAccessScopeStore

	lock sync.RWMutex
}

func (ds *dataStoreImpl) GetRole(ctx context.Context, name string) (*storage.Role, bool, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}

	return ds.roleStorage.Get(ctx, name)
}

func (ds *dataStoreImpl) GetAllRoles(ctx context.Context) ([]*storage.Role, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	return ds.getAllRolesNoScopeCheck(ctx)
}

func (ds *dataStoreImpl) getAllRolesNoScopeCheck(ctx context.Context) ([]*storage.Role, error) {
	var roles []*storage.Role
	err := ds.roleStorage.Walk(ctx, func(role *storage.Role) error {
		roles = append(roles, role)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return roles, nil
}

func (ds *dataStoreImpl) AddRole(ctx context.Context, role *storage.Role) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := rolePkg.ValidateRole(role); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := verifyNotDefaultRole(role.GetName()); err != nil {
		return err
	}

	// protect against TOCTOU race condition
	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyRoleNameDoesNotExist(ctx, role.GetName()); err != nil {
		return err
	}
	if err := ds.verifyRoleReferencesExist(ctx, role); err != nil {
		return err
	}

	return ds.roleStorage.Upsert(ctx, role)
}

func (ds *dataStoreImpl) UpdateRole(ctx context.Context, role *storage.Role) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := rolePkg.ValidateRole(role); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := verifyNotDefaultRole(role.GetName()); err != nil {
		return err
	}

	// protect against TOCTOU race condition
	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyRoleNameExists(ctx, role.GetName()); err != nil {
		return err
	}
	if err := ds.verifyRoleReferencesExist(ctx, role); err != nil {
		return err
	}

	return ds.roleStorage.Upsert(ctx, role)
}

func (ds *dataStoreImpl) RemoveRole(ctx context.Context, name string) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := verifyNotDefaultRole(name); err != nil {
		return err
	}
	// Verify storage constraints.
	if err := ds.verifyRoleNameExists(ctx, name); err != nil {
		return err
	}

	return ds.roleStorage.Delete(ctx, name)
}

////////////////////////////////////////////////////////////////////////////////
// Permission sets                                                            //
//                                                                            //

func (ds *dataStoreImpl) GetPermissionSet(ctx context.Context, id string) (*storage.PermissionSet, bool, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}

	return ds.permissionSetStorage.Get(ctx, id)
}

func (ds *dataStoreImpl) GetAllPermissionSets(ctx context.Context) ([]*storage.PermissionSet, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var permissionSets []*storage.PermissionSet
	err := ds.permissionSetStorage.Walk(ctx, func(permissionSet *storage.PermissionSet) error {
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
	if err := rolePkg.ValidatePermissionSet(permissionSet); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := verifyNotDefaultPermissionSet(permissionSet.GetName()); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyPermissionSetIDDoesNotExist(ctx, permissionSet.GetId()); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no permission set with the same name.
	if err := ds.permissionSetStorage.Upsert(ctx, permissionSet); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) UpdatePermissionSet(ctx context.Context, permissionSet *storage.PermissionSet) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := rolePkg.ValidatePermissionSet(permissionSet); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := verifyNotDefaultPermissionSet(permissionSet.GetName()); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyPermissionSetIDExists(ctx, permissionSet.GetId()); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no permission set with the same name.
	if err := ds.permissionSetStorage.Upsert(ctx, permissionSet); err != nil {
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

	permissionSet, found, err := ds.permissionSetStorage.Get(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return errors.Wrapf(errox.NotFound, "id = %s", id)
	}
	if err := verifyNotDefaultPermissionSet(permissionSet.GetName()); err != nil {
		return err
	}

	// Ensure this PermissionSet isn't in use by any Role.
	roles, err := ds.getAllRolesNoScopeCheck(ctx)
	if err != nil {
		return err
	}
	for _, role := range roles {
		if role.GetPermissionSetId() == id {
			return errors.Wrapf(errox.ReferencedByAnotherObject, "cannot delete permission set in use by role %q", role.GetName())
		}
	}

	// Constraints ok, delete the object.
	if err := ds.permissionSetStorage.Delete(ctx, id); err != nil {
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

	return ds.accessScopeStorage.Get(ctx, id)
}

func (ds *dataStoreImpl) GetAllAccessScopes(ctx context.Context) ([]*storage.SimpleAccessScope, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var scopes []*storage.SimpleAccessScope
	err := ds.accessScopeStorage.Walk(ctx, func(scope *storage.SimpleAccessScope) error {
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
	if err := rolePkg.ValidateSimpleAccessScope(scope); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := verifyNotDefaultAccessScope(scope); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyAccessScopeIDDoesNotExist(ctx, scope.GetId()); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no access scope with the same name.
	if err := ds.accessScopeStorage.Upsert(ctx, scope); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) UpdateAccessScope(ctx context.Context, scope *storage.SimpleAccessScope) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := rolePkg.ValidateSimpleAccessScope(scope); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := verifyNotDefaultAccessScope(scope); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	if err := ds.verifyAccessScopeIDExists(ctx, scope.GetId()); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no access scope with the same name.
	if err := ds.accessScopeStorage.Upsert(ctx, scope); err != nil {
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
	accessScope, found, err := ds.accessScopeStorage.Get(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return errors.Wrapf(errox.NotFound, "id = %s", id)
	}
	if err := verifyNotDefaultAccessScope(accessScope); err != nil {
		return err
	}

	// Ensure this AccessScope isn't in use by any Role.
	roles, err := ds.getAllRolesNoScopeCheck(ctx)
	if err != nil {
		return err
	}
	for _, role := range roles {
		if role.GetAccessScopeId() == id {
			return errors.Wrapf(errox.ReferencedByAnotherObject, "cannot delete access scope in use by role %q", role.GetName())
		}
	}

	// Constraints ok, delete the object.
	if err := ds.accessScopeStorage.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) GetAndResolveRole(ctx context.Context, name string) (permissions.ResolvedRole, error) {
	if ok, err := roleSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	ds.lock.RLock()
	defer ds.lock.RUnlock()

	// No need to continue if the role does not exist.
	role, found, err := ds.roleStorage.Get(ctx, name)
	if err != nil || !found {
		return nil, err
	}

	permissionSet, err := ds.getRolePermissionSetOrError(ctx, role)
	if err != nil {
		return nil, err
	}

	accessScope, err := ds.getRoleAccessScopeOrError(ctx, role)
	if err != nil {
		return nil, err
	}

	resolvedRole := &resolvedRoleImpl{
		role:          role,
		permissionSet: permissionSet,
		accessScope:   accessScope,
	}

	return resolvedRole, nil
}

////////////////////////////////////////////////////////////////////////////////
// Storage constraints                                                        //
//                                                                            //
// Uniqueness of the 'name' field is expected to be verified by the           //
// underlying store, see its `--uniq-key-func` flag                           //

func (ds *dataStoreImpl) verifyRoleReferencesExist(ctx context.Context, role *storage.Role) error {
	// Verify storage constraints.
	if err := ds.verifyPermissionSetIDExists(ctx, role.GetPermissionSetId()); err != nil {
		return errors.Wrapf(errox.InvalidArgs, "referenced permission set %s does not exist", role.GetPermissionSetId())
	}
	if err := ds.verifyAccessScopeIDExists(ctx, role.GetAccessScopeId()); err != nil {
		return errors.Wrapf(errox.InvalidArgs, "referenced access scope %s does not exist", role.GetAccessScopeId())
	}
	return nil
}

// Returns errox.InvalidArgs if the given role is a default one.
func verifyNotDefaultRole(name string) error {
	if rolePkg.IsDefaultRoleName(name) {
		return errors.Wrapf(errox.InvalidArgs, "default role %q cannot be modified or deleted", name)
	}
	return nil
}

// Returns errox.NotFound if there is no permission set with the supplied ID.
func (ds *dataStoreImpl) verifyPermissionSetIDExists(ctx context.Context, id string) error {
	_, found, err := ds.permissionSetStorage.Get(ctx, id)

	if err != nil {
		return err
	}
	if !found {
		return errors.Wrapf(errox.NotFound, "id = %s", id)
	}
	return nil
}

// Returns errox.AlreadyExists if there is a permission set with the same ID.
func (ds *dataStoreImpl) verifyPermissionSetIDDoesNotExist(ctx context.Context, id string) error {
	_, found, err := ds.permissionSetStorage.Get(ctx, id)

	if err != nil {
		return err
	}
	if found {
		return errors.Wrapf(errox.AlreadyExists, "id = %s", id)
	}
	return nil
}

// Returns errox.InvalidArgs if the given permission set is a default
// one. Note that IsDefaultRoleName() is reused due to the name sameness.
func verifyNotDefaultPermissionSet(name string) error {
	if rolePkg.IsDefaultRoleName(name) {
		return errors.Wrapf(errox.InvalidArgs, "default permission set %q cannot be modified or deleted", name)
	}
	return nil
}

// Returns errox.NotFound if there is no access scope with the supplied ID.
func (ds *dataStoreImpl) verifyAccessScopeIDExists(ctx context.Context, id string) error {
	_, found, err := ds.accessScopeStorage.Get(ctx, id)

	if err != nil {
		return err
	}
	if !found {
		return errors.Wrapf(errox.NotFound, "id = %s", id)
	}
	return nil
}

// Returns errox.AlreadyExists if there is an access scope with the same ID.
func (ds *dataStoreImpl) verifyAccessScopeIDDoesNotExist(ctx context.Context, id string) error {
	_, found, err := ds.accessScopeStorage.Get(ctx, id)

	if err != nil {
		return err
	}
	if found {
		return errors.Wrapf(errox.AlreadyExists, "id = %s", id)
	}
	return nil
}

// Returns errox.AlreadyExists if there is a role with the same name.
func (ds *dataStoreImpl) verifyRoleNameDoesNotExist(ctx context.Context, name string) error {
	_, found, err := ds.roleStorage.Get(ctx, name)

	if err != nil {
		return err
	}
	if found {
		return errors.Wrapf(errox.AlreadyExists, "name = %q", name)
	}
	return nil
}

// Returns errox.NotFound if there is no role with the supplied name.
func (ds *dataStoreImpl) verifyRoleNameExists(ctx context.Context, name string) error {
	_, found, err := ds.roleStorage.Get(ctx, name)

	if err != nil {
		return err
	}
	if !found {
		return errors.Wrapf(errox.NotFound, "name = %q", name)
	}
	return nil
}

// Returns errox.InvalidArgs if the given scope is a default one.
func verifyNotDefaultAccessScope(scope *storage.SimpleAccessScope) error {
	if rolePkg.IsDefaultAccessScope(scope.GetId()) {
		return errors.Wrapf(errox.InvalidArgs, "default access scope %q cannot be modified or deleted", scope.GetName())
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Helpers                                                                    //
//                                                                            //

// Finds the permission set associated with the given role. Every stored role
// must reference an existing permission set.
func (ds *dataStoreImpl) getRolePermissionSetOrError(ctx context.Context, role *storage.Role) (*storage.PermissionSet, error) {
	permissionSet, found, err := ds.permissionSetStorage.Get(ctx, role.GetPermissionSetId())
	if err != nil {
		return nil, err
	} else if !found || permissionSet == nil {
		log.Errorf("Failed to fetch permission set %s for the existing role %q", role.GetPermissionSetId(), role.GetName())
		return nil, errors.Wrapf(errox.InvariantViolation, "permission set %s for role %q is missing", role.GetPermissionSetId(), role.GetName())
	}
	return permissionSet, nil
}

// Finds the access scope associated with the given role. Every stored role must
// reference an existing access scope.
func (ds *dataStoreImpl) getRoleAccessScopeOrError(ctx context.Context, role *storage.Role) (*storage.SimpleAccessScope, error) {
	accessScope, found, err := ds.accessScopeStorage.Get(ctx, role.GetAccessScopeId())
	if err != nil {
		return nil, err
	} else if !found || accessScope == nil {
		log.Errorf("Failed to fetch access scope %s for the existing role %q", role.GetAccessScopeId(), role.GetName())
		return nil, errors.Wrapf(errox.InvariantViolation, "access scope %s for role %q is missing", role.GetAccessScopeId(), role.GetName())
	}
	return accessScope, nil
}
