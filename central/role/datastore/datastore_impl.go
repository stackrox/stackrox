package datastore

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	roleSAC = sac.ForResource(resources.Access)

	log = logging.LoggerForModule()
)

type dataStoreImpl struct {
	roleStorage          store.RoleStore
	permissionSetStorage store.PermissionSetStore
	accessScopeStorage   store.SimpleAccessScopeStore
	groupGetFilteredFunc func(ctx context.Context, filter func(*storage.Group) bool) ([]*storage.Group, error)

	lock sync.RWMutex
}

func (ds *dataStoreImpl) UpsertRole(ctx context.Context, newRole *storage.Role) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := rolePkg.ValidateRole(newRole); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	oldRole, exists, err := ds.roleStorage.Get(ctx, newRole.GetName())
	if err != nil {
		return err
	}
	if exists {
		if err := verifyRoleOrigin(ctx, oldRole); err != nil {
			return err
		}
	}
	if err := verifyRoleOrigin(ctx, newRole); err != nil {
		return err
	}

	if err := ds.verifyRoleReferencesExist(ctx, newRole); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no role with the same name.
	if err := ds.roleStorage.Upsert(ctx, newRole); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) UpsertPermissionSet(ctx context.Context, newPS *storage.PermissionSet) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := rolePkg.ValidatePermissionSet(newPS); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	oldPS, exists, err := ds.permissionSetStorage.Get(ctx, newPS.GetId())
	if err != nil {
		return err
	}
	if exists {
		if err := verifyPermissionSetOrigin(ctx, oldPS); err != nil {
			return err
		}
	}
	if err := verifyPermissionSetOrigin(ctx, newPS); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no permission set with the same name.
	if err := ds.permissionSetStorage.Upsert(ctx, newPS); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) UpsertAccessScope(ctx context.Context, newScope *storage.SimpleAccessScope) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := rolePkg.ValidateSimpleAccessScope(newScope); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	oldScope, exists, err := ds.accessScopeStorage.Get(ctx, newScope.GetId())
	if err != nil {
		return err
	}
	if exists {
		if err := verifyAccessScopeOrigin(ctx, oldScope); err != nil {
			return err
		}
	}
	if err := verifyAccessScopeOrigin(ctx, newScope); err != nil {
		return err
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no access scope with the same name.
	if err := ds.accessScopeStorage.Upsert(ctx, newScope); err != nil {
		return err
	}

	return nil
}

func (ds *dataStoreImpl) GetRole(ctx context.Context, name string) (*storage.Role, bool, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return nil, false, err
	}
	return ds.roleStorage.Get(ctx, name)
}

func (ds *dataStoreImpl) GetAllRoles(ctx context.Context) ([]*storage.Role, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	return ds.getAllRolesNoScopeCheck(ctx)
}

func (ds *dataStoreImpl) GetRolesFiltered(ctx context.Context, filter func(role *storage.Role) bool) ([]*storage.Role, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}

	var filteredRoles []*storage.Role
	walkFn := func() error {
		filteredRoles = filteredRoles[:0]
		return ds.roleStorage.Walk(ctx, func(role *storage.Role) error {
			if filter(role) {
				filteredRoles = append(filteredRoles, role)
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return filteredRoles, nil
}

func (ds *dataStoreImpl) CountRoles(ctx context.Context) (int, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return 0, err
	}

	return ds.roleStorage.Count(ctx)
}

func (ds *dataStoreImpl) getAllRolesNoScopeCheck(ctx context.Context) ([]*storage.Role, error) {
	var roles []*storage.Role
	walkFn := func() error {
		roles = roles[:0]
		return ds.roleStorage.Walk(ctx, func(role *storage.Role) error {
			roles = append(roles, role)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
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
	if err := verifyNotDefaultRole(role); err != nil {
		return err
	}
	if err := verifyRoleOrigin(ctx, role); err != nil {
		return errors.Wrap(err, "origin didn't match for role")
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
	if err := verifyNotDefaultRole(role); err != nil {
		return err
	}

	// protect against TOCTOU race condition
	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	existingRole, err := ds.verifyRoleNameExists(ctx, role.GetName())
	if err != nil {
		return err
	}
	if err = verifyRoleOrigin(ctx, existingRole); err != nil {
		return errors.Wrap(err, "origin didn't match for existing role")
	}
	if err = verifyRoleOrigin(ctx, role); err != nil {
		return errors.Wrap(err, "origin didn't match for new role")
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

	if err := ds.verifyRoleForDeletion(ctx, name); err != nil {
		return err
	}

	return ds.roleStorage.Delete(ctx, name)
}

func verifyRoleOrigin(ctx context.Context, role *storage.Role) error {
	if !declarativeconfig.CanModifyResource(ctx, role) {
		return errors.Wrapf(errox.NotAuthorized, "role %q's origin is %s, cannot be modified or deleted with the current permission",
			role.GetName(), role.GetTraits().GetOrigin())
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Permission sets                                                            //
//                                                                            //

func (ds *dataStoreImpl) GetPermissionSet(ctx context.Context, id string) (*storage.PermissionSet, bool, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return nil, false, err
	}
	return ds.permissionSetStorage.Get(ctx, id)
}

func (ds *dataStoreImpl) GetAllPermissionSets(ctx context.Context) ([]*storage.PermissionSet, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	var permissionSets []*storage.PermissionSet
	walkFn := func() error {
		permissionSets = permissionSets[:0]
		return ds.permissionSetStorage.Walk(ctx, func(permissionSet *storage.PermissionSet) error {
			permissionSets = append(permissionSets, permissionSet)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}

	return permissionSets, nil
}

func (ds *dataStoreImpl) GetPermissionSetsFiltered(ctx context.Context,
	filter func(permissionSet *storage.PermissionSet) bool) ([]*storage.PermissionSet, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	var filteredPermissionSets []*storage.PermissionSet
	walkFn := func() error {
		filteredPermissionSets = filteredPermissionSets[:0]
		return ds.permissionSetStorage.Walk(ctx, func(permissionSet *storage.PermissionSet) error {
			if filter(permissionSet) {
				filteredPermissionSets = append(filteredPermissionSets, permissionSet)
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}

	return filteredPermissionSets, nil
}

func (ds *dataStoreImpl) CountPermissionSets(ctx context.Context) (int, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return 0, err
	}
	return ds.permissionSetStorage.Count(ctx)
}

func (ds *dataStoreImpl) AddPermissionSet(ctx context.Context, permissionSet *storage.PermissionSet) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := rolePkg.ValidatePermissionSet(permissionSet); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := verifyNotDefaultPermissionSet(permissionSet); err != nil {
		return err
	}

	if err := verifyPermissionSetOrigin(ctx, permissionSet); err != nil {
		return errors.Wrap(err, "origin didn't match for new permission set")
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
	if err := verifyNotDefaultPermissionSet(permissionSet); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	existingPermissionSet, err := ds.verifyPermissionSetIDExists(ctx, permissionSet.GetId())
	if err != nil {
		return err
	}
	if err := verifyPermissionSetOrigin(ctx, existingPermissionSet); err != nil {
		return errors.Wrap(err, "origin didn't match for existing permission set")
	}
	if err := verifyPermissionSetOrigin(ctx, permissionSet); err != nil {
		return errors.Wrap(err, "origin didn't match for new permission set")
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
	if err := verifyNotDefaultPermissionSet(permissionSet); err != nil {
		return err
	}
	if err := verifyPermissionSetOrigin(ctx, permissionSet); err != nil {
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

func verifyPermissionSetOrigin(ctx context.Context, ps *storage.PermissionSet) error {
	if !declarativeconfig.CanModifyResource(ctx, ps) {
		return errors.Wrapf(errox.NotAuthorized, "permission set %q's origin is %s, cannot be modified or deleted with the current permission",
			ps.GetName(), ps.GetTraits().GetOrigin())
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Access scopes                                                              //
//                                                                            //

func (ds *dataStoreImpl) GetAccessScope(ctx context.Context, id string) (*storage.SimpleAccessScope, bool, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return nil, false, err
	}
	return ds.accessScopeStorage.Get(ctx, id)
}

func (ds *dataStoreImpl) GetAllAccessScopes(ctx context.Context) ([]*storage.SimpleAccessScope, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	var scopes []*storage.SimpleAccessScope
	walkFn := func() error {
		scopes = scopes[:0]
		return ds.accessScopeStorage.Walk(ctx, func(scope *storage.SimpleAccessScope) error {
			scopes = append(scopes, scope)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}

	return scopes, nil
}

func (ds *dataStoreImpl) GetAccessScopesFiltered(ctx context.Context,
	filter func(accessScope *storage.SimpleAccessScope) bool) ([]*storage.SimpleAccessScope, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return nil, err
	}
	var filteredScopes []*storage.SimpleAccessScope
	walkFn := func() error {
		filteredScopes = filteredScopes[:0]
		return ds.accessScopeStorage.Walk(ctx, func(scope *storage.SimpleAccessScope) error {
			if filter(scope) {
				filteredScopes = append(filteredScopes, scope)
			}
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}

	return filteredScopes, nil
}

func (ds *dataStoreImpl) CountAccessScopes(ctx context.Context) (int, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return 0, err
	}
	return ds.accessScopeStorage.Count(ctx)
}

func (ds *dataStoreImpl) AccessScopeExists(ctx context.Context, id string) (bool, error) {
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
		return false, err
	}
	found, err := ds.accessScopeStorage.Exists(ctx, id)
	if err != nil || !found {
		return false, err
	}
	return true, nil
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

	if err := verifyAccessScopeOrigin(ctx, scope); err != nil {
		return errors.Wrap(err, "origin didn't match for new access scope")
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

func (ds *dataStoreImpl) UpdateAccessScope(ctx context.Context, newScope *storage.SimpleAccessScope) error {
	if err := sac.VerifyAuthzOK(roleSAC.WriteAllowed(ctx)); err != nil {
		return err
	}
	if err := rolePkg.ValidateSimpleAccessScope(newScope); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := verifyNotDefaultAccessScope(newScope); err != nil {
		return err
	}

	ds.lock.Lock()
	defer ds.lock.Unlock()

	// Verify storage constraints.
	existingScope, err := ds.verifyAccessScopeIDExists(ctx, newScope.GetId())
	if err != nil {
		return err
	}
	if err := verifyAccessScopeOrigin(ctx, existingScope); err != nil {
		return errors.Wrap(err, "origin didn't match for existing access scope")
	}
	if err := verifyAccessScopeOrigin(ctx, newScope); err != nil {
		return errors.Wrap(err, "origin didn't match for new access scope")
	}

	// Constraints ok, write the object. We expect the underlying store to
	// verify there is no access scope with the same name.
	if err := ds.accessScopeStorage.Upsert(ctx, newScope); err != nil {
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
	if err := verifyAccessScopeOrigin(ctx, accessScope); err != nil {
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
	if err := sac.VerifyAuthzOK(roleSAC.ReadAllowed(ctx)); err != nil {
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

func (ds *dataStoreImpl) GetAllResolvedRoles(ctx context.Context) ([]permissions.ResolvedRole, error) {
	roles, err := ds.GetAllRoles(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list all roles")
	}
	permissionSets, err := ds.GetAllPermissionSets(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list all permission sets")
	}
	accessScopes, err := ds.GetAllAccessScopes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list all access scopes")
	}
	permissionSetsByID := make(map[string]*storage.PermissionSet, len(permissionSets))
	for _, ps := range permissionSets {
		permissionSetsByID[ps.GetId()] = ps
	}
	accessScopesByID := make(map[string]*storage.SimpleAccessScope, len(accessScopes))
	for _, as := range accessScopes {
		accessScopesByID[as.GetId()] = as
	}
	result := make([]permissions.ResolvedRole, 0, len(roles))
	for _, role := range roles {
		resolvedRole := &resolvedRoleImpl{
			role: role,
		}
		if ps, ok := permissionSetsByID[role.GetPermissionSetId()]; ok {
			resolvedRole.permissionSet = ps
		} else {
			return nil, errors.Wrapf(errox.InvariantViolation, "no permission set found for role %s", role.GetName())
		}
		if as, ok := accessScopesByID[role.GetAccessScopeId()]; ok {
			resolvedRole.accessScope = as
		} else {
			return nil, errors.Wrapf(errox.InvariantViolation, "no access scope found for role %s", role.GetName())
		}
		result = append(result, resolvedRole)
	}
	return result, nil
}

func verifyAccessScopeOrigin(ctx context.Context, as *storage.SimpleAccessScope) error {
	if !declarativeconfig.CanModifyResource(ctx, as) {
		return errors.Wrapf(errox.NotAuthorized, "access scope %q's origin is %s, cannot be modified or deleted with the current permission",
			as.GetName(), as.GetTraits().GetOrigin())
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Storage constraints                                                        //
//                                                                            //
// Uniqueness of the 'name' field is expected to be verified by the           //
// underlying store, see its `--uniq-key-func` flag                           //

func (ds *dataStoreImpl) verifyRoleReferencesExist(ctx context.Context, role *storage.Role) error {
	// Verify storage constraints.
	permissionSet, err := ds.verifyPermissionSetIDExists(ctx, role.GetPermissionSetId())
	if err != nil {
		return errors.Wrapf(errox.InvalidArgs, "referenced permission set %s does not exist", role.GetPermissionSetId())
	}
	accessScope, err := ds.verifyAccessScopeIDExists(ctx, role.GetAccessScopeId())
	if err != nil {
		return errors.Wrapf(errox.InvalidArgs, "referenced access scope %s does not exist", role.GetAccessScopeId())
	}

	if err := declarativeconfig.VerifyReferencedResourceOrigin(permissionSet, role, permissionSet.GetName(), role.GetName()); err != nil {
		return err
	}
	if err := declarativeconfig.VerifyReferencedResourceOrigin(accessScope, role, accessScope.GetName(), role.GetName()); err != nil {
		return err
	}

	return nil
}

// Returns errox.InvalidArgs if the given role is a default one.
func verifyNotDefaultRole(role *storage.Role) error {
	if rolePkg.IsDefaultRole(role) {
		return errors.Wrapf(errox.InvalidArgs, "default role %q cannot be modified or deleted", role.GetName())
	}
	return nil
}

// Returns errox.NotFound if there is no permission set with the supplied ID.
func (ds *dataStoreImpl) verifyPermissionSetIDExists(ctx context.Context, id string) (*storage.PermissionSet, error) {
	ps, found, err := ds.permissionSetStorage.Get(ctx, id)

	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "id = %s", id)
	}
	return ps, nil
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
// one. Note that IsDefaultRole() is reused due to the name sameness.
func verifyNotDefaultPermissionSet(permissionSet *storage.PermissionSet) error {
	if rolePkg.IsDefaultPermissionSet(permissionSet) {
		return errors.Wrapf(errox.InvalidArgs, "default permission set %q cannot be modified or deleted",
			permissionSet.GetName())
	}
	return nil
}

// Returns errox.NotFound if there is no access scope with the supplied ID.
func (ds *dataStoreImpl) verifyAccessScopeIDExists(ctx context.Context, id string) (*storage.SimpleAccessScope, error) {
	as, found, err := ds.accessScopeStorage.Get(ctx, id)

	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "id = %s", id)
	}
	return as, nil
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
func (ds *dataStoreImpl) verifyRoleNameExists(ctx context.Context, name string) (*storage.Role, error) {
	role, found, err := ds.roleStorage.Get(ctx, name)

	if err != nil {
		return nil, err
	}
	if !found {
		return nil, errors.Wrapf(errox.NotFound, "name = %q", name)
	}
	return role, nil
}

// verifyRoleForDeletion verifies the storage constraints for deleting a role.
// It will:
// - verify that the role is not a default role
// - verify that the role exists
func (ds *dataStoreImpl) verifyRoleForDeletion(ctx context.Context, name string) error {
	role, found, err := ds.roleStorage.Get(ctx, name)

	if err != nil {
		return err
	}
	if !found {
		return errors.Wrapf(errox.NotFound, "name = %q", name)
	}
	if err := verifyRoleOrigin(ctx, role); err != nil {
		return err
	}

	if err := verifyNotDefaultRole(role); err != nil {
		return err
	}

	return ds.verifyNoGroupReferences(ctx, role)
}

// Returns errox.ReferencedByAnotherObject if the given role is referenced by a group.
func (ds *dataStoreImpl) verifyNoGroupReferences(ctx context.Context, role *storage.Role) error {
	groups, err := ds.groupGetFilteredFunc(ctx, func(group *storage.Group) bool {
		return group.GetRoleName() == role.GetName()
	})
	if err != nil {
		return err
	}

	if len(groups) > 0 {
		return errox.ReferencedByAnotherObject.Newf("role %s is referenced by groups [%s] in auth providers, "+
			"ensure all references to the role are removed", role.GetName(), strings.Join(getGroupIDs(groups), ","))
	}
	return nil
}

// Returns errox.InvalidArgs if the given scope is a default one.
func verifyNotDefaultAccessScope(scope *storage.SimpleAccessScope) error {
	if rolePkg.IsDefaultAccessScope(scope) {
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

func getGroupIDs(groups []*storage.Group) []string {
	groupIDs := make([]string, 0, len(groups))
	for _, group := range groups {
		groupIDs = append(groupIDs, group.GetProps().GetId())
	}
	return groupIDs
}
