package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/role"
	roleStore "github.com/stackrox/rox/central/role/datastore/internal/store"
	"github.com/stackrox/rox/central/role/resources"
	permissionSetStore "github.com/stackrox/rox/central/role/store/permissionset/rocksdb"
	simpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestAllDefaultRolesAreCovered(t *testing.T) {
	assert.Len(t, defaultRoles, len(role.DefaultRoleNames))
	for r := range defaultRoles {
		assert.Contains(t, role.DefaultRoleNames, r)
	}
}

func TestRoleDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(roleDataStoreTestSuite))
}

// Note that the access scope and permission set tests deviate from the testing
// style used by the role tests: instead of using store's mock, an instance of
// the underlying storage layer (rocksdb) is created. We do not really care
// about how the validation logic is split between the access scope datastore
// and the underlying rocksdb CRUD layer, but we verify if the datastore as a
// whole behaves as expected.
type roleDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx  context.Context
	hasReadCtx  context.Context
	hasWriteCtx context.Context

	dataStore DataStore
	boltDB    *bolt.DB
	rocksie   *rocksdb.RocksDB

	existingRole          *storage.Role
	existingPermissionSet *storage.PermissionSet
	existingScope         *storage.SimpleAccessScope
}

func (s *roleDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Role)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Role)))

	s.initDataStore(true)
}

func (s *roleDataStoreTestSuite) initDataStore(useRolesWithPermissionSets bool) {
	var err error
	s.boltDB, err = bolthelper.NewTemp(s.T().Name() + "-bolt.db")
	s.Require().NoError(err)
	s.rocksie = rocksdbtest.RocksDBForT(s.T())

	roleStorage := roleStore.New(s.boltDB)
	permissionSetStorage, err := permissionSetStore.New(s.rocksie)
	s.Require().NoError(err)
	scopeStorage, err := simpleAccessScopeStore.New(s.rocksie)
	s.Require().NoError(err)

	s.dataStore = New(roleStorage, permissionSetStorage, scopeStorage, useRolesWithPermissionSets)

	if useRolesWithPermissionSets {
		// Insert a permission set, access scope, and role into the test DB.
		s.existingPermissionSet = getValidPermissionSet("permissionset.existing", "existing permissionset")
		s.Require().NoError(permissionSetStorage.Upsert(s.existingPermissionSet))
		s.existingScope = getValidAccessScope("scope.existing", "existing scope")
		s.Require().NoError(scopeStorage.Upsert(s.existingScope))
		s.existingRole = getValidRole("existing role", s.existingPermissionSet.GetId(), s.existingScope.GetId())
		s.Require().NoError(roleStorage.AddRole(s.existingRole))
	} else {
		s.existingRole = getValidRole("existing role", "", "")
		s.existingRole.ResourceToAccess = map[string]storage.Access{
			"Policy": storage.Access_READ_ACCESS,
		}
		s.Require().NoError(roleStorage.AddRole(s.existingRole))
	}
}

// This will go away when we sunset old role format.
func (s *roleDataStoreTestSuite) reInitDataStore(useRolesWithPermissionSets bool) {
	s.TearDownTest()
	s.initDataStore(useRolesWithPermissionSets)
}

func (s *roleDataStoreTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.rocksie)
	testutils.TearDownDB(s.boltDB)
}

////////////////////////////////////////////////////////////////////////////////
// Roles                                                                      //
//                                                                            //

func getValidRole(name, permissionSetID, accessScopeID string) *storage.Role {
	return &storage.Role{
		Name:            name,
		PermissionSetId: permissionSetID,
		AccessScopeId:   accessScopeID,
	}
}

func getInvalidRole(name string) *storage.Role {
	return &storage.Role{
		Name:            name,
		PermissionSetId: "some non-existent permission set",
		AccessScopeId:   "some non-existent scope",
	}
}

func (s *roleDataStoreTestSuite) TestRolePermissions() {
	// goodRole and badRole should validate but do not actually exist in the database.
	goodRole := getValidRole("new valid role", s.existingPermissionSet.GetId(), s.existingScope.GetId())
	badRole := getInvalidRole("new invalid role")

	role, err := s.dataStore.GetRole(s.hasNoneCtx, s.existingRole.GetName())
	s.NoError(err, "no access for Get*() is not an error")
	s.Nil(role)

	role, err = s.dataStore.GetRole(s.hasNoneCtx, goodRole.GetName())
	s.NoError(err, "no error even if the object does not exist")
	s.Nil(role)

	roles, err := s.dataStore.GetAllRoles(s.hasNoneCtx)
	s.NoError(err, "no access for GetAll*() is not an error")
	s.Empty(roles)

	err = s.dataStore.AddRole(s.hasNoneCtx, goodRole)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for Add*() yields a permission error")

	err = s.dataStore.AddRole(s.hasReadCtx, goodRole)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "READ access for Add*() yields a permission error")

	err = s.dataStore.AddRole(s.hasReadCtx, badRole)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error for invalid Role")

	err = s.dataStore.UpdateRole(s.hasNoneCtx, s.existingRole)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for Update*() yields a permission error")

	err = s.dataStore.UpdateRole(s.hasReadCtx, s.existingRole)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "READ access for Update*() yields a permission error")

	err = s.dataStore.UpdateRole(s.hasReadCtx, goodRole)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error if the object does not exist")

	err = s.dataStore.RemoveRole(s.hasNoneCtx, s.existingRole.GetName())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for Remove*() yields a permission error")

	err = s.dataStore.RemoveRole(s.hasReadCtx, s.existingRole.GetName())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "READ access for Remove*() yields a permission error")

	err = s.dataStore.RemoveRole(s.hasReadCtx, goodRole.GetName())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error if the object does not exist")
}

////////////////////////////////////////////////////////////////////////////////
// Permission sets                                                            //
//                                                                            //

func getValidPermissionSet(id string, name string) *storage.PermissionSet {
	return &storage.PermissionSet{
		Id:   role.EnsureValidPermissionSetID(id),
		Name: name,
	}
}

func getInvalidPermissionSet(id string, name string) *storage.PermissionSet {
	return &storage.PermissionSet{
		Id:   role.EnsureValidPermissionSetID(id),
		Name: name,
		ResourceToAccess: map[string]storage.Access{
			"Some non-existent resource": storage.Access_NO_ACCESS,
		},
	}
}

func (s *roleDataStoreTestSuite) TestPermissionSetPermissions() {
	goodPermissionSet := getValidPermissionSet("permissionset.valid", "new valid permission set")
	badPermissionSet := getInvalidPermissionSet("permissionset.invalid", "new invalid permission set")

	permissionSet, found, err := s.dataStore.GetPermissionSet(s.hasNoneCtx, s.existingPermissionSet.GetId())
	s.NoError(err, "no access for Get*() is not an error")
	s.False(found)
	s.Nil(permissionSet)

	permissionSet, found, err = s.dataStore.GetPermissionSet(s.hasNoneCtx, goodPermissionSet.GetId())
	s.NoError(err, "no error even if the object does not exist")
	s.False(found)
	s.Nil(permissionSet)

	permissionSets, err := s.dataStore.GetAllPermissionSets(s.hasNoneCtx)
	s.NoError(err, "no access for Get*() is not an error")
	s.Empty(permissionSets)

	err = s.dataStore.AddPermissionSet(s.hasNoneCtx, goodPermissionSet)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for Add*() yields a permission error")

	err = s.dataStore.AddPermissionSet(s.hasReadCtx, goodPermissionSet)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "READ access for Add*() yields a permission error")

	err = s.dataStore.AddPermissionSet(s.hasReadCtx, badPermissionSet)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error for invalid permissionSet")

	err = s.dataStore.UpdatePermissionSet(s.hasNoneCtx, s.existingPermissionSet)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for Update*() yields a permission error")

	err = s.dataStore.UpdatePermissionSet(s.hasReadCtx, s.existingPermissionSet)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "READ access for Update*() yields a permission error")

	err = s.dataStore.UpdatePermissionSet(s.hasReadCtx, goodPermissionSet)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error if the object does not exist")

	err = s.dataStore.RemovePermissionSet(s.hasNoneCtx, s.existingPermissionSet.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for Remove*() yields a permission error")

	err = s.dataStore.RemovePermissionSet(s.hasReadCtx, s.existingPermissionSet.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "READ access for Remove*() yields a permission error")

	err = s.dataStore.RemovePermissionSet(s.hasReadCtx, goodPermissionSet.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error if the object does not exist")
}

func (s *roleDataStoreTestSuite) TestPermissionSetReadOperations() {
	misplacedPermissionSet := getValidPermissionSet("permissionset.misplaced", "non-existing permission set")

	permissionSet, found, err := s.dataStore.GetPermissionSet(s.hasReadCtx, misplacedPermissionSet.GetId())
	s.NoError(err, "not found for Get*() is not an error")
	s.False(found)
	s.Nil(permissionSet)

	permissionSet, found, err = s.dataStore.GetPermissionSet(s.hasReadCtx, s.existingPermissionSet.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(s.existingPermissionSet, permissionSet, "with READ access existing object is returned")

	permissionSets, err := s.dataStore.GetAllPermissionSets(s.hasReadCtx)
	s.NoError(err)
	s.Len(permissionSets, 1, "with READ access all objects are returned")
}

func (s *roleDataStoreTestSuite) TestPermissionSetWriteOperations() {
	goodPermissionSet := getValidPermissionSet("permissionset.new", "new valid permissionset")
	badPermissionSet := getInvalidPermissionSet("permissionset.new", "new invalid permissionset")
	mimicPermissionSet := getValidPermissionSet("permissionset.new", "existing permissionset")
	clonePermissionSet := getValidPermissionSet("permissionset.existing", "new existing permissionset")
	updatedAdminPermissionSet := getValidPermissionSet(role.EnsureValidAccessScopeID("admin"), role.Admin)

	err := s.dataStore.AddPermissionSet(s.hasWriteCtx, badPermissionSet)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "invalid permission set for Add*() yields an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, clonePermissionSet)
	s.ErrorIs(err, errorhelpers.ErrAlreadyExists, "adding permission set with an existing ID yields an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, mimicPermissionSet)
	s.ErrorIs(err, errorhelpers.ErrAlreadyExists, "adding permission set with an existing name yields an error")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, goodPermissionSet)
	s.ErrorIs(err, errorhelpers.ErrNotFound, "updating non-existing permission set yields an error")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, updatedAdminPermissionSet)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "updating a default permission set yields an error")

	err = s.dataStore.RemovePermissionSet(s.hasWriteCtx, goodPermissionSet.GetId())
	s.ErrorIs(err, errorhelpers.ErrNotFound, "removing non-existing permission set yields an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, goodPermissionSet)
	s.NoError(err)

	permissionSets, _ := s.dataStore.GetAllPermissionSets(s.hasReadCtx)
	s.Len(permissionSets, 2, "added permission set should be visible in the subsequent Get*()")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, badPermissionSet)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "invalid permission set for Update*() yields an error")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, mimicPermissionSet)
	s.ErrorIs(err, errorhelpers.ErrAlreadyExists, "introducing a name collision with Update*() yields an error")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, goodPermissionSet)
	s.NoError(err)

	err = s.dataStore.RemovePermissionSet(s.hasWriteCtx, goodPermissionSet.GetId())
	s.NoError(err)

	permissionSets, _ = s.dataStore.GetAllPermissionSets(s.hasReadCtx)
	s.Len(permissionSets, 1, "removed permission set should be absent in the subsequent Get*()")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, goodPermissionSet)
	s.NoError(err, "adding a permission set with ID and name that used to exist is not an error")
}

////////////////////////////////////////////////////////////////////////////////
// Access scopes                                                              //
//                                                                            //

func getValidAccessScope(id string, name string) *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id:   role.EnsureValidAccessScopeID(id),
		Name: name,
	}
}

func getInvalidAccessScope(id string, name string) *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id:   role.EnsureValidAccessScopeID(id),
		Name: name,
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				{
					ClusterName: "missing namespace name",
				},
			},
		},
	}
}

func (s *roleDataStoreTestSuite) TestAccessScopePermissions() {
	goodScope := getValidAccessScope("scope.valid", "new valid scope")
	badScope := getInvalidAccessScope("scope.invalid", "new invalid scope")

	scope, found, err := s.dataStore.GetAccessScope(s.hasNoneCtx, s.existingScope.GetId())
	s.NoError(err, "no access for Get*() is not an error")
	s.False(found)
	s.Nil(scope)

	scope, found, err = s.dataStore.GetAccessScope(s.hasNoneCtx, goodScope.GetId())
	s.NoError(err, "no error even if the object does not exist")
	s.False(found)
	s.Nil(scope)

	scopes, err := s.dataStore.GetAllAccessScopes(s.hasNoneCtx)
	s.NoError(err, "no access for Get*() is not an error")
	s.Empty(scopes)

	err = s.dataStore.AddAccessScope(s.hasNoneCtx, goodScope)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for Add*() yields a permission error")

	err = s.dataStore.AddAccessScope(s.hasReadCtx, goodScope)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "READ access for Add*() yields a permission error")

	err = s.dataStore.AddAccessScope(s.hasReadCtx, badScope)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error for invalid scope")

	err = s.dataStore.UpdateAccessScope(s.hasNoneCtx, s.existingScope)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for Update*() yields a permission error")

	err = s.dataStore.UpdateAccessScope(s.hasReadCtx, s.existingScope)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "READ access for Update*() yields a permission error")

	err = s.dataStore.UpdateAccessScope(s.hasReadCtx, goodScope)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error if the object does not exist")

	err = s.dataStore.RemoveAccessScope(s.hasNoneCtx, s.existingScope.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for Remove*() yields a permission error")

	err = s.dataStore.RemoveAccessScope(s.hasReadCtx, s.existingScope.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "READ access for Remove*() yields a permission error")

	err = s.dataStore.RemoveAccessScope(s.hasReadCtx, goodScope.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error if the object does not exist")
}

func (s *roleDataStoreTestSuite) TestAccessScopeReadOperations() {
	misplacedScope := getValidAccessScope("scope.misplaced", "non-existing scope")

	scope, found, err := s.dataStore.GetAccessScope(s.hasReadCtx, misplacedScope.GetId())
	s.NoError(err, "not found for Get*() is not an error")
	s.False(found)
	s.Nil(scope)

	scope, found, err = s.dataStore.GetAccessScope(s.hasReadCtx, s.existingScope.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(s.existingScope, scope, "with READ access existing object is returned")

	scopes, err := s.dataStore.GetAllAccessScopes(s.hasReadCtx)
	s.NoError(err)
	s.Len(scopes, 1, "with READ access all objects are returned")
}

func (s *roleDataStoreTestSuite) TestAccessScopeWriteOperations() {
	goodScope := getValidAccessScope("scope.new", "new valid scope")
	badScope := getInvalidAccessScope("scope.new", "new invalid scope")
	mimicScope := getValidAccessScope("scope.new", "existing scope")
	cloneScope := getValidAccessScope("scope.existing", "new existing scope")
	updatedDefaultScope := getValidAccessScope("io.stackrox.authz.accessscope.denyall", role.AccessScopeExcludeAll.GetName())

	err := s.dataStore.AddAccessScope(s.hasWriteCtx, badScope)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "invalid scope for Add*() yields an error")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, cloneScope)
	s.ErrorIs(err, errorhelpers.ErrAlreadyExists, "adding scope with an existing ID yields an error")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, mimicScope)
	s.ErrorIs(err, errorhelpers.ErrAlreadyExists, "adding scope with an existing name yields an error")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, goodScope)
	s.ErrorIs(err, errorhelpers.ErrNotFound, "updating non-existing scope yields an error")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, updatedDefaultScope)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "updating a default scope yields an error")

	err = s.dataStore.RemoveAccessScope(s.hasWriteCtx, goodScope.GetId())
	s.ErrorIs(err, errorhelpers.ErrNotFound, "removing non-existing scope yields an error")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, goodScope)
	s.NoError(err)

	scopes, _ := s.dataStore.GetAllAccessScopes(s.hasReadCtx)
	s.Len(scopes, 2, "added scope should be visible in the subsequent Get*()")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, badScope)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "invalid scope for Update*() yields an error")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, mimicScope)
	s.ErrorIs(err, errorhelpers.ErrAlreadyExists, "introducing a name collision with Update*() yields an error")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, goodScope)
	s.NoError(err)

	err = s.dataStore.RemoveAccessScope(s.hasWriteCtx, goodScope.GetId())
	s.NoError(err)

	scopes, _ = s.dataStore.GetAllAccessScopes(s.hasReadCtx)
	s.Len(scopes, 1, "removed scope should be absent in the subsequent Get*()")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, goodScope)
	s.NoError(err, "adding a scope with ID and name that used to exist is not an error")
}

////////////////////////////////////////////////////////////////////////////////
// Combined                                                                   //
//                                                                            //

func (s *roleDataStoreTestSuite) TestForeignKeyConstraints() {
	var err error
	permissionSet := getValidPermissionSet("permissionset.new", "new valid permissionset")
	scope := getValidAccessScope("scope.new", "new valid scope")
	role := getValidRole("new valid role", permissionSet.GetId(), scope.GetId())

	err = s.dataStore.AddRole(s.hasWriteCtx, role)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "Cannot create a Role without its PermissionSet and AccessScope existing")

	s.NoError(s.dataStore.AddPermissionSet(s.hasWriteCtx, permissionSet))

	err = s.dataStore.AddRole(s.hasWriteCtx, role)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "Cannot create a Role without its AccessScope existing")

	s.NoError(s.dataStore.AddAccessScope(s.hasWriteCtx, scope))
	s.NoError(s.dataStore.AddRole(s.hasWriteCtx, role))

	err = s.dataStore.RemovePermissionSet(s.hasWriteCtx, permissionSet.GetId())
	s.ErrorIs(err, errorhelpers.ErrReferencedByAnotherObject, "cannot delete a PermissionSet referred to by a Role")

	err = s.dataStore.RemoveAccessScope(s.hasWriteCtx, scope.GetId())
	s.ErrorIs(err, errorhelpers.ErrReferencedByAnotherObject, "cannot delete an Access Scope referred to by a Role")

	s.NoError(s.dataStore.RemoveRole(s.hasWriteCtx, role.GetName()))
	s.NoError(s.dataStore.RemovePermissionSet(s.hasWriteCtx, permissionSet.GetId()))
	s.NoError(s.dataStore.RemoveAccessScope(s.hasWriteCtx, scope.GetId()))
}

func (s *roleDataStoreTestSuite) TestGetAndResolveRoleNewFormat() {
	noScopeRole := getValidRole("role without a scope", s.existingPermissionSet.GetId(), "")

	resolvedRole, err := s.dataStore.GetAndResolveRole(s.hasNoneCtx, s.existingRole.GetName())
	s.NoError(err, "no access for GetAndResolveRole() is not an error")
	s.Nil(resolvedRole)

	resolvedRole, err = s.dataStore.GetAndResolveRole(s.hasReadCtx, noScopeRole.GetName())
	s.NoError(err, "no error even if the role does not exist")
	s.Nil(resolvedRole)

	err = s.dataStore.AddRole(s.hasWriteCtx, noScopeRole)
	s.NoError(err)
	resolvedRole, err = s.dataStore.GetAndResolveRole(s.hasReadCtx, noScopeRole.GetName())
	s.NoError(err, "no error if the role does not reference a scope")
	s.Equal(noScopeRole.GetName(), resolvedRole.GetRoleName())
	s.Equal(s.existingPermissionSet.GetResourceToAccess(), resolvedRole.GetPermissions())
	s.Nil(resolvedRole.GetAccessScope())

	resolvedRole, err = s.dataStore.GetAndResolveRole(s.hasReadCtx, s.existingRole.GetName())
	s.NoError(err)
	s.Equal(s.existingRole.GetName(), resolvedRole.GetRoleName())
	s.Equal(s.existingPermissionSet.GetResourceToAccess(), resolvedRole.GetPermissions())
	s.Equal(s.existingScope, resolvedRole.GetAccessScope())
}

func (s *roleDataStoreTestSuite) TestGetAndResolveRoleOldFormat() {
	s.reInitDataStore(false)

	misplacedRole := getValidRole("non-existing role", "", "")

	resolvedRole, err := s.dataStore.GetAndResolveRole(s.hasNoneCtx, s.existingRole.GetName())
	s.NoError(err, "no access for GetAndResolveRole() is not an error")
	s.Nil(resolvedRole)

	resolvedRole, err = s.dataStore.GetAndResolveRole(s.hasReadCtx, misplacedRole.GetName())
	s.NoError(err, "no error even if the role does not exist")
	s.Nil(resolvedRole)

	resolvedRole, err = s.dataStore.GetAndResolveRole(s.hasReadCtx, s.existingRole.GetName())
	s.NoError(err)
	s.Equal(s.existingRole.GetName(), resolvedRole.GetRoleName())
	s.Equal(s.existingRole.GetResourceToAccess(), resolvedRole.GetPermissions())
}

func (s *roleDataStoreTestSuite) TestValidateRoleNewFormat() {
	err := s.dataStore.AddRole(s.hasWriteCtx, &storage.Role{})
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "name field must be set")

	err = s.dataStore.AddRole(s.hasWriteCtx, &storage.Role{
		Name: "name",
		ResourceToAccess: map[string]storage.Access{
			"Policy": storage.Access_READ_ACCESS,
		},
	})
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "role must not have resourceToAccess field set")

	noPermsRole := getValidRole("role with no permission set", "", "")
	err = s.dataStore.AddRole(s.hasWriteCtx, noPermsRole)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "role must reference an existing permission set")

	updatedAdminRole := getValidRole(role.Admin, s.existingPermissionSet.GetId(), "")
	err = s.dataStore.UpdateRole(s.hasWriteCtx, updatedAdminRole)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "updating a default role yields an error")

	noScopeRole := getValidRole("role with no access scope", s.existingPermissionSet.GetId(), "")
	err = s.dataStore.AddRole(s.hasWriteCtx, noScopeRole)
	s.NoError(err, "empty access scope reference is allowed")

	goodRole := getValidRole("new valid role", s.existingPermissionSet.GetId(), s.existingScope.GetId())
	err = s.dataStore.AddRole(s.hasWriteCtx, goodRole)
	s.NoError(err)
}

func (s *roleDataStoreTestSuite) TestValidateRoleOldFormat() {
	s.reInitDataStore(false)

	roleWithPermSet := getValidRole("role with permission set", "some permissionset", "")
	err := s.dataStore.AddRole(s.hasWriteCtx, roleWithPermSet)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "permission sets are not supported in the old role format")

	roleWithScope := getValidRole("role with scope", "", "some accessscope")
	err = s.dataStore.AddRole(s.hasWriteCtx, roleWithScope)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "access scope are not supported in the old role format")

	roleWithInvalidResource := &storage.Role{
		Name: "name",
		ResourceToAccess: map[string]storage.Access{
			"EndlessSummer": storage.Access_READ_WRITE_ACCESS,
		},
	}
	err = s.dataStore.AddRole(s.hasWriteCtx, roleWithInvalidResource)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "non-existing resources are not supported")

	goodRole := &storage.Role{
		Name: "new valid role",
		ResourceToAccess: map[string]storage.Access{
			"Policy": storage.Access_READ_ACCESS,
		},
	}
	err = s.dataStore.AddRole(s.hasWriteCtx, goodRole)
	s.NoError(err)
}
