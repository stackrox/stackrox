//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/role"
	PermissionSetPGStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	postgresRolePGStore "github.com/stackrox/rox/central/role/store/role/postgres"
	postgresSimpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestAllDefaultRolesAreCovered(t *testing.T) {
	// Merge the roles for vuln reporting into the defaults
	defaultRoles := getDefaultRoles()
	assert.Len(t, defaultRoles, len(accesscontrol.DefaultRoleNames))
	for _, r := range defaultRoles {
		assert.Truef(t, accesscontrol.DefaultRoleNames.Contains(r.GetName()), "role %s not found in default role names", r)
	}
}

func TestEachDefaultRoleHasDefaultPermSet(t *testing.T) {
	for _, role := range getDefaultRoles() {
		permSet := getDefaultPermissionSet(role.GetName())
		assert.NotNil(t, permSet)
		assert.Equal(t, permSet.GetId(), role.GetPermissionSetId())
	}
}

func TestAnalystPermSetDoesNotContainAdministration(t *testing.T) {
	analystPermSet, found := defaultPermissionSets[accesscontrol.Analyst]
	// Analyst is one of the default roles.
	assert.True(t, found)

	resourceToAccess := analystPermSet.resourceWithAccess
	// Contains all resources except one.
	assert.Len(t, resourceToAccess, len(resources.ListAll())-1)
	// Does not contain Administration resource.
	for _, resource := range resourceToAccess {
		assert.NotEqual(t, resource.Resource.GetResource(), resources.Administration.GetResource())
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

	hasNoneCtx             context.Context
	hasReadCtx             context.Context
	hasWriteCtx            context.Context
	hasWriteDeclarativeCtx context.Context

	dataStore    DataStore
	postgresTest *pgtest.TestPostgres

	existingRole                     *storage.Role
	existingPermissionSet            *storage.PermissionSet
	existingScope                    *storage.SimpleAccessScope
	existingDeclarativePermissionSet *storage.PermissionSet
	existingDeclarativeScope         *storage.SimpleAccessScope

	filteredFuncReturnValue []*storage.Group
	filteredFuncReturnError error
}

func (s *roleDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeyList(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
	s.hasWriteDeclarativeCtx = declarativeconfig.WithModifyDeclarativeResource(s.hasWriteCtx)

	s.initDataStore()
}

func (s *roleDataStoreTestSuite) mockGroupGetFiltered(_ context.Context, _ func(*storage.Group) bool) ([]*storage.Group, error) {
	return s.filteredFuncReturnValue, s.filteredFuncReturnError
}

func (s *roleDataStoreTestSuite) initDataStore() {
	s.postgresTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.postgresTest)

	roleStorage := postgresRolePGStore.New(s.postgresTest.DB)
	permissionSetStorage := PermissionSetPGStore.New(s.postgresTest.DB)
	accessScopeStorage := postgresSimpleAccessScopeStore.New(s.postgresTest.DB)

	s.dataStore = New(roleStorage, permissionSetStorage, accessScopeStorage, s.mockGroupGetFiltered)

	// Insert a permission set, access scope, and role into the test DB.
	s.existingPermissionSet = getValidPermissionSet("permissionset.existing", "existing permissionset")
	s.Require().NoError(permissionSetStorage.Upsert(s.hasWriteCtx, s.existingPermissionSet))
	s.existingScope = getValidAccessScope("scope.existing", "existing scope")
	s.Require().NoError(accessScopeStorage.Upsert(s.hasWriteCtx, s.existingScope))
	s.existingRole = getValidRole("existing role", s.existingPermissionSet.GetId(), s.existingScope.GetId())
	s.Require().NoError(roleStorage.Upsert(s.hasWriteCtx, s.existingRole))

	// Insert declarative permission set and access scope to reference by declarative roles.
	s.existingDeclarativePermissionSet = getValidPermissionSet("permissionset.existing.declarative", "existing declarative permissionset")
	s.existingDeclarativePermissionSet.Traits = &storage.Traits{
		Origin: storage.Traits_DECLARATIVE,
	}
	s.Require().NoError(permissionSetStorage.Upsert(s.hasWriteDeclarativeCtx, s.existingDeclarativePermissionSet))
	s.existingDeclarativeScope = getValidAccessScope("scope.existing.declarative", "existing declarative scope")
	s.existingDeclarativeScope.Traits = &storage.Traits{
		Origin: storage.Traits_DECLARATIVE,
	}
	s.Require().NoError(accessScopeStorage.Upsert(s.hasWriteDeclarativeCtx, s.existingDeclarativeScope))

}

func (s *roleDataStoreTestSuite) TearDownTest() {
	s.postgresTest.Close()
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

	role, found, err := s.dataStore.GetRole(s.hasNoneCtx, s.existingRole.GetName())
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.False(found, "not found")
	s.Nil(role)

	role, found, err = s.dataStore.GetRole(s.hasNoneCtx, goodRole.GetName())
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.False(found, "not found")
	s.Nil(role)

	roles, err := s.dataStore.GetAllRoles(s.hasNoneCtx)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
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

func (s *roleDataStoreTestSuite) TestRoleReadOperations() {
	role, found, err := s.dataStore.GetRole(s.hasReadCtx, "non-existing role")
	s.NoError(err, "not found for Get*() is not an error")
	s.False(found)
	s.Nil(role)

	role, found, err = s.dataStore.GetRole(s.hasReadCtx, s.existingRole.GetName())
	s.NoError(err)
	s.True(found)
	s.Equal(s.existingRole, role, "with READ access existing object is returned")

	roles, err := s.dataStore.GetAllRoles(s.hasReadCtx)
	s.NoError(err)
	s.Len(roles, 1, "with READ access all objects are returned")

	roles, err = s.dataStore.GetRolesFiltered(s.hasReadCtx, func(role *storage.Role) bool {
		return role.GetName() == s.existingRole.GetName()
	})
	s.NoError(err)
	s.Len(roles, 1)
	s.ElementsMatch(roles, []*storage.Role{role})

	roles, err = s.dataStore.GetRolesFiltered(s.hasReadCtx, func(role *storage.Role) bool {
		return role.GetName() == "non-existing-role"
	})
	s.NoError(err)
	s.Empty(roles)
}

func (s *roleDataStoreTestSuite) TestRoleWriteOperations() {
	goodRole := getValidRole("valid role", s.existingPermissionSet.GetId(), s.existingScope.GetId())
	secondExistingPermissionSet := getValidPermissionSet("permissionset.existingtoo", "second permission set")
	updatedGoodRole := getValidRole("valid role", secondExistingPermissionSet.GetId(), s.existingScope.GetId())
	badRole := &storage.Role{Name: "invalid role"}
	cloneRole := getValidRole(s.existingRole.GetName(), s.existingPermissionSet.GetId(), s.existingScope.GetId())
	updatedAdminRole := getValidRole(accesscontrol.Admin, s.existingPermissionSet.GetId(), s.existingScope.GetId())
	declarativeRole := getValidRole("declarative role", s.existingDeclarativePermissionSet.GetId(), s.existingDeclarativeScope.GetId())
	declarativeRole.Traits = &storage.Traits{
		Origin: storage.Traits_DECLARATIVE,
	}
	badDeclarativeRole := getInvalidRole("invalid declarative role")
	badDeclarativeRole.Traits = &storage.Traits{
		Origin: storage.Traits_DECLARATIVE,
	}
	s.setupGetFilteredReturnValues([]*storage.Group{}, nil)

	err := s.dataStore.AddPermissionSet(s.hasWriteCtx, secondExistingPermissionSet)
	s.NoError(err, "failed to add second permission set needed for test")

	err = s.dataStore.AddRole(s.hasWriteCtx, badRole)
	s.ErrorIs(err, errox.InvalidArgs, "invalid role for Add*() yields an error")

	err = s.dataStore.AddRole(s.hasWriteCtx, cloneRole)
	s.ErrorIs(err, errox.AlreadyExists, "adding role with an existing name yields an error")

	err = s.dataStore.UpdateRole(s.hasWriteCtx, goodRole)
	s.ErrorIs(err, errox.NotFound, "updating non-existing role yields an error")

	err = s.dataStore.UpdateRole(s.hasWriteCtx, updatedAdminRole)
	s.ErrorIs(err, errox.InvalidArgs, "updating a default role yields an error")

	err = s.dataStore.RemoveRole(s.hasWriteCtx, goodRole.GetName())
	s.ErrorIs(err, errox.NotFound, "removing non-existing role yields an error")

	err = s.dataStore.AddRole(s.hasWriteDeclarativeCtx, goodRole)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to add declaratively imperative role is an error")

	err = s.dataStore.AddRole(s.hasWriteCtx, goodRole)
	s.NoError(err)

	roles, _ := s.dataStore.GetAllRoles(s.hasReadCtx)
	s.Len(roles, 2, "added roles should be visible in the subsequent Get*()")

	err = s.dataStore.UpdateRole(s.hasWriteCtx, badRole)
	s.ErrorIs(err, errox.InvalidArgs, "invalid role for Update*() yields an error")

	err = s.dataStore.UpdateRole(s.hasWriteCtx, updatedGoodRole)
	s.NoError(err)

	datastoreGoodRole, found, err := s.dataStore.GetRole(s.hasReadCtx, goodRole.GetName())
	s.NoError(err)
	s.True(found)
	s.Equal(updatedGoodRole.GetPermissionSetId(), datastoreGoodRole.GetPermissionSetId(),
		"successful Update*() call should update the value in datastore")

	err = s.dataStore.RemoveRole(s.hasWriteCtx, goodRole.GetName())
	s.NoError(err)

	roles, _ = s.dataStore.GetAllRoles(s.hasReadCtx)
	s.Len(roles, 1, "removed role should be absent in the subsequent Get*()")

	err = s.dataStore.AddRole(s.hasWriteCtx, goodRole)
	s.NoError(err, "adding a role with name that used to exist is not an error")

	err = s.dataStore.AddRole(s.hasWriteCtx, declarativeRole)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to add imperatively declarative role is an error")

	err = s.dataStore.AddRole(s.hasWriteDeclarativeCtx, declarativeRole)
	s.NoError(err, "adding a declarative role with declarative context is not an error")

	err = s.dataStore.UpdateRole(s.hasWriteCtx, declarativeRole)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to modify imperatively declarative role is an error")

	err = s.dataStore.UpdateRole(s.hasWriteDeclarativeCtx, goodRole)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to modify declaratively imperative role is an error")

	err = s.dataStore.UpdateRole(s.hasWriteDeclarativeCtx, declarativeRole)
	s.NoError(err, "attempting to modify declaratively declarative role is not an error")

	err = s.dataStore.RemoveRole(s.hasWriteCtx, declarativeRole.GetName())
	s.ErrorIs(err, errox.NotAuthorized, "attempting to delete imperatively declarative role is an error")

	err = s.dataStore.RemoveRole(s.hasWriteDeclarativeCtx, goodRole.GetName())
	s.ErrorIs(err, errox.NotAuthorized, "attempting to delete declaratively imperative role is an error")

	err = s.dataStore.RemoveRole(s.hasWriteDeclarativeCtx, declarativeRole.GetName())
	s.NoError(err, "attempting to delete declaratively declarative role is not an error")

	err = s.dataStore.UpsertRole(s.hasWriteCtx, declarativeRole)
	s.ErrorIs(err, errox.NotAuthorized, "upserting imperatively declarative role is an error")

	err = s.dataStore.UpsertRole(s.hasWriteDeclarativeCtx, declarativeRole)
	s.NoError(err, "attempting to upsert declaratively declarative role is not an error")

	err = s.dataStore.UpsertRole(s.hasWriteDeclarativeCtx, declarativeRole)
	s.NoError(err, "re-upserting declaratively declarative role is not an error")

	err = s.dataStore.UpsertRole(s.hasWriteCtx, declarativeRole)
	s.ErrorIs(err, errox.NotAuthorized, "re-upserting imperatively declarative role is an error")

	err = s.dataStore.UpsertRole(s.hasWriteDeclarativeCtx, goodRole)
	s.ErrorIs(err, errox.NotAuthorized, "upserting declaratively imperative role is an error")

	err = s.dataStore.UpsertRole(s.hasWriteCtx, goodRole)
	s.NoError(err, "attempting to upsert imperatively imperative role is not an error")

	err = s.dataStore.UpsertRole(s.hasWriteCtx, goodRole)
	s.NoError(err, "re-upserting imperatively imperative role is not an error")

	err = s.dataStore.UpsertRole(s.hasWriteDeclarativeCtx, goodRole)
	s.ErrorIs(err, errox.NotAuthorized, "re-upserting declaratively imperative role is an error")

	err = s.dataStore.UpsertRole(s.hasWriteCtx, badRole)
	s.ErrorIs(err, errox.InvalidArgs, "invalid scope for Upsert*() yields an error(imperative resource)")

	err = s.dataStore.UpsertRole(s.hasWriteDeclarativeCtx, badDeclarativeRole)
	s.ErrorIs(err, errox.InvalidArgs, "invalid scope for Upsert*() yields an error(declarative resource)")
}

func (s *roleDataStoreTestSuite) setupGetFilteredReturnValues(groups []*storage.Group, err error) {
	s.filteredFuncReturnValue = groups
	s.filteredFuncReturnError = err
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
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.False(found)
	s.Nil(permissionSet)

	permissionSet, found, err = s.dataStore.GetPermissionSet(s.hasNoneCtx, goodPermissionSet.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.False(found)
	s.Nil(permissionSet)

	permissionSets, err := s.dataStore.GetAllPermissionSets(s.hasNoneCtx)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
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
	s.Len(permissionSets, 2, "with READ access all objects are returned")

	permissionSets, err = s.dataStore.GetPermissionSetsFiltered(s.hasReadCtx, func(permissionSet *storage.PermissionSet) bool {
		return permissionSet.GetId() == s.existingPermissionSet.GetId()
	})
	s.NoError(err)
	s.Len(permissionSets, 1)
	s.ElementsMatch(permissionSets, []*storage.PermissionSet{s.existingPermissionSet})

	permissionSets, err = s.dataStore.GetPermissionSetsFiltered(s.hasReadCtx, func(permissionSet *storage.PermissionSet) bool {
		return permissionSet.GetId() == "non-existing permission set"
	})
	s.NoError(err)
	s.Empty(permissionSets)
}

func (s *roleDataStoreTestSuite) TestPermissionSetWriteOperations() {
	goodPermissionSet := getValidPermissionSet("permissionset.new", "new valid permissionset")
	updatedGoodPermissionSet := goodPermissionSet.Clone()
	updatedGoodPermissionSet.ResourceToAccess = map[string]storage.Access{
		resources.Namespace.String(): storage.Access_READ_WRITE_ACCESS,
	}
	badPermissionSet := getInvalidPermissionSet("permissionset.new", "new invalid permissionset")
	mimicPermissionSet := &storage.PermissionSet{
		Id:   goodPermissionSet.Id,
		Name: "existing permissionset",
	}
	clonePermissionSet := &storage.PermissionSet{
		Id:   s.existingPermissionSet.Id,
		Name: "new existing permissionset",
	}
	declarativePermissionSet := getValidPermissionSet("permissionset.declarative", "declarative permissionset")
	declarativePermissionSet.Traits = &storage.Traits{
		Origin: storage.Traits_DECLARATIVE,
	}
	badDeclarativePermissionSet := getInvalidPermissionSet("permissionset.declarative.invalid", "invalid declarative role")
	badDeclarativePermissionSet.Traits = &storage.Traits{
		Origin: storage.Traits_DECLARATIVE,
	}
	updatedAdminPermissionSet := getValidPermissionSet(role.EnsureValidAccessScopeID("admin"), accesscontrol.Admin)

	err := s.dataStore.AddPermissionSet(s.hasWriteCtx, badPermissionSet)
	s.ErrorIs(err, errox.InvalidArgs, "invalid permission set for Add*() yields an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, clonePermissionSet)
	s.ErrorIs(err, errox.AlreadyExists, "adding permission set with an existing ID yields an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, mimicPermissionSet)
	// With postgres the unique constraint catches this.
	assert.ErrorContains(s.T(), err, "violates unique constraint")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, goodPermissionSet)
	s.ErrorIs(err, errox.NotFound, "updating non-existing permission set yields an error")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, updatedAdminPermissionSet)
	s.ErrorIs(err, errox.InvalidArgs, "updating a default permission set yields an error")

	err = s.dataStore.RemovePermissionSet(s.hasWriteCtx, goodPermissionSet.GetId())
	s.ErrorIs(err, errox.NotFound, "removing non-existing permission set yields an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteDeclarativeCtx, goodPermissionSet)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to add declaratively imperative permission set is an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, goodPermissionSet)
	s.NoError(err)

	permissionSets, _ := s.dataStore.GetAllPermissionSets(s.hasReadCtx)
	s.Len(permissionSets, 3, "added permission set should be visible in the subsequent Get*()")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, badPermissionSet)
	s.ErrorIs(err, errox.InvalidArgs, "invalid permission set for Update*() yields an error")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, mimicPermissionSet)
	// With postgres the unique constraint catches this.
	assert.ErrorContains(s.T(), err, "violates unique constraint")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, updatedGoodPermissionSet)
	s.NoError(err)

	datastoreGoodPermissionSet, found, err := s.dataStore.GetPermissionSet(s.hasReadCtx, goodPermissionSet.GetId())
	s.NoError(err)
	s.True(found)
	namespaceAccess, ok := datastoreGoodPermissionSet.GetResourceToAccess()[resources.Namespace.String()]
	s.True(ok)
	s.Equal(storage.Access_READ_WRITE_ACCESS, namespaceAccess,
		"successful Update*() call should update the value in datastore")

	err = s.dataStore.RemovePermissionSet(s.hasWriteCtx, goodPermissionSet.GetId())
	s.NoError(err)

	permissionSets, _ = s.dataStore.GetAllPermissionSets(s.hasReadCtx)
	s.Len(permissionSets, 2, "removed permission set should be absent in the subsequent Get*()")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, goodPermissionSet)
	s.NoError(err, "adding a permission set with ID and name that used to exist is not an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, declarativePermissionSet)
	s.ErrorIs(err, errox.NotAuthorized, "adding a declarative permission set imperatively is an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteDeclarativeCtx, declarativePermissionSet)
	s.NoError(err, "adding a declarative permission set declaratively is not an error")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, declarativePermissionSet)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to modify imperatively declarative permission set is an error")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteDeclarativeCtx, goodPermissionSet)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to modify declaratively imperative permission set is an error")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteDeclarativeCtx, declarativePermissionSet)
	s.NoError(err, "attempting to modify declaratively declarative permission set is not an error")

	err = s.dataStore.RemovePermissionSet(s.hasWriteCtx, declarativePermissionSet.GetId())
	s.ErrorIs(err, errox.NotAuthorized, "attempting to delete imperatively declarative permission set is an error")

	err = s.dataStore.RemovePermissionSet(s.hasWriteDeclarativeCtx, goodPermissionSet.GetId())
	s.ErrorIs(err, errox.NotAuthorized, "attempting to delete declaratively imperative permission set is an error")

	err = s.dataStore.RemovePermissionSet(s.hasWriteDeclarativeCtx, declarativePermissionSet.GetId())
	s.NoError(err, "attempting to delete declaratively declarative permission set is not an error")

	s.Len(permissionSets, 2, "removed permission set should be absent in the subsequent Get*()")

	err = s.dataStore.UpsertPermissionSet(s.hasWriteCtx, declarativePermissionSet)
	s.ErrorIs(err, errox.NotAuthorized, "upserting imperatively declarative role is an error")

	err = s.dataStore.UpsertPermissionSet(s.hasWriteDeclarativeCtx, declarativePermissionSet)
	s.NoError(err, "attempting to upsert declaratively declarative role is not an error")

	err = s.dataStore.UpsertPermissionSet(s.hasWriteDeclarativeCtx, declarativePermissionSet)
	s.NoError(err, "re-upserting declaratively declarative role is not an error")

	err = s.dataStore.UpsertPermissionSet(s.hasWriteCtx, declarativePermissionSet)
	s.ErrorIs(err, errox.NotAuthorized, "re-upserting imperatively declarative role is an error")

	err = s.dataStore.UpsertPermissionSet(s.hasWriteDeclarativeCtx, goodPermissionSet)
	s.ErrorIs(err, errox.NotAuthorized, "upserting declaratively imperative role is an error")

	err = s.dataStore.UpsertPermissionSet(s.hasWriteCtx, goodPermissionSet)
	s.NoError(err, "attempting to upsert imperatively imperative role is not an error")

	err = s.dataStore.UpsertPermissionSet(s.hasWriteCtx, goodPermissionSet)
	s.NoError(err, "re-upserting imperatively imperative role is not an error")

	err = s.dataStore.UpsertPermissionSet(s.hasWriteDeclarativeCtx, goodPermissionSet)
	s.ErrorIs(err, errox.NotAuthorized, "re-upserting declaratively imperative role is an error")

	err = s.dataStore.UpsertPermissionSet(s.hasWriteCtx, badPermissionSet)
	s.ErrorIs(err, errox.InvalidArgs, "invalid scope for Upsert*() yields an error(imperative resource)")

	err = s.dataStore.UpsertPermissionSet(s.hasWriteDeclarativeCtx, badDeclarativePermissionSet)
	s.ErrorIs(err, errox.InvalidArgs, "invalid scope for Upsert*() yields an error(declarative resource)")
}

////////////////////////////////////////////////////////////////////////////////
// Access scopes                                                              //
//                                                                            //

func getValidAccessScope(id string, name string) *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id:    role.EnsureValidAccessScopeID(id),
		Name:  name,
		Rules: &storage.SimpleAccessScope_Rules{},
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
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.False(found)
	s.Nil(scope)

	scope, found, err = s.dataStore.GetAccessScope(s.hasNoneCtx, goodScope.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.False(found)
	s.Nil(scope)

	scopes, err := s.dataStore.GetAllAccessScopes(s.hasNoneCtx)
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.Empty(scopes)

	exists, err := s.dataStore.AccessScopeExists(s.hasNoneCtx, s.existingScope.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for AcessScopeExists yields a permission error")
	s.False(exists)

	exists, err = s.dataStore.AccessScopeExists(s.hasNoneCtx, goodScope.GetId())
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error if the object does not exist")
	s.False(exists)

	err = s.dataStore.AddAccessScope(s.hasNoneCtx, goodScope)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "no access for Add*() yields a permission error")

	err = s.dataStore.AddAccessScope(s.hasReadCtx, goodScope)
	s.ErrorIs(err, sac.ErrResourceAccessDenied, "still a permission error for invalid scope")

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

	exists, err := s.dataStore.AccessScopeExists(s.hasReadCtx, misplacedScope.GetId())
	s.NoError(err, "not existing scope for AccessScopeExists() should not return error")
	s.False(exists)

	exists, err = s.dataStore.AccessScopeExists(s.hasReadCtx, s.existingScope.GetId())
	s.NoError(err)
	s.True(exists)

	scopes, err := s.dataStore.GetAllAccessScopes(s.hasReadCtx)
	s.NoError(err)
	s.Len(scopes, 2, "with READ access all objects are returned")

	scopes, err = s.dataStore.GetAccessScopesFiltered(s.hasReadCtx, func(accessScope *storage.SimpleAccessScope) bool {
		return accessScope.GetId() == s.existingScope.GetId()
	})
	s.NoError(err)
	s.Len(scopes, 1)
	s.ElementsMatch(scopes, []*storage.SimpleAccessScope{s.existingScope})

	scopes, err = s.dataStore.GetAccessScopesFiltered(s.hasReadCtx, func(accessScope *storage.SimpleAccessScope) bool {
		return accessScope.GetId() == "non-existing scope"
	})
	s.NoError(err)
	s.Empty(scopes)
}

func (s *roleDataStoreTestSuite) TestAccessScopeWriteOperations() {
	goodScope := getValidAccessScope("scope.new", "new valid scope")
	updatedGoodScope := goodScope.Clone()
	updatedIncludedClusters := []string{"clusterA"}
	updatedGoodScope.Rules = &storage.SimpleAccessScope_Rules{
		IncludedClusters: updatedIncludedClusters,
	}
	badScope := getInvalidAccessScope("scope.new", "new invalid scope")
	mimicScope := &storage.SimpleAccessScope{
		Id:    goodScope.Id,
		Name:  "existing scope",
		Rules: &storage.SimpleAccessScope_Rules{},
	}
	cloneScope := &storage.SimpleAccessScope{
		Id:    s.existingScope.Id,
		Name:  "new existing scope",
		Rules: &storage.SimpleAccessScope_Rules{},
	}
	updatedDefaultScope := getValidAccessScope("ffffffff-ffff-fff4-f5ff-fffffffffffe",
		role.AccessScopeExcludeAll.GetName())
	declarativeScope := getValidAccessScope("scope.declarative", "new declarative scope")
	declarativeScope.Traits = &storage.Traits{
		Origin: storage.Traits_DECLARATIVE,
	}
	badDeclarativeScope := getInvalidAccessScope("scope.declarative-invalid", "new invalid declarative scope")
	badDeclarativeScope.Traits = &storage.Traits{
		Origin: storage.Traits_DECLARATIVE,
	}

	err := s.dataStore.AddAccessScope(s.hasWriteCtx, badScope)
	s.ErrorIs(err, errox.InvalidArgs, "invalid scope for Add*() yields an error")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, cloneScope)
	s.ErrorIs(err, errox.AlreadyExists, "adding scope with an existing ID yields an error")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, mimicScope)
	// With postgres the unique constraint catches this.
	assert.ErrorContains(s.T(), err, "violates unique constraint")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, goodScope)
	s.ErrorIs(err, errox.NotFound, "updating non-existing scope yields an error")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, updatedDefaultScope)
	s.ErrorIs(err, errox.InvalidArgs, "updating a default scope yields an error")

	err = s.dataStore.RemoveAccessScope(s.hasWriteCtx, goodScope.GetId())
	s.ErrorIs(err, errox.NotFound, "removing non-existing scope yields an error")

	err = s.dataStore.AddAccessScope(s.hasWriteDeclarativeCtx, goodScope)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to modify declaratively imperative access scope is an error")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, goodScope)
	s.NoError(err)

	scopes, _ := s.dataStore.GetAllAccessScopes(s.hasReadCtx)
	s.Len(scopes, 3, "added scope should be visible in the subsequent Get*()")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, badScope)
	s.ErrorIs(err, errox.InvalidArgs, "invalid scope for Update*() yields an error")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, mimicScope)
	// With postgres the unique constraint catches this.
	assert.ErrorContains(s.T(), err, "violates unique constraint")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, updatedGoodScope)
	s.NoError(err)

	datastoreGoodAccessScope, found, err := s.dataStore.GetAccessScope(s.hasReadCtx, updatedGoodScope.GetId())
	s.NoError(err)
	s.True(found)
	s.Equal(updatedIncludedClusters, datastoreGoodAccessScope.GetRules().GetIncludedClusters(),
		"successful Update*() call should update the value in datastore")

	err = s.dataStore.RemoveAccessScope(s.hasWriteCtx, goodScope.GetId())
	s.NoError(err)

	scopes, _ = s.dataStore.GetAllAccessScopes(s.hasReadCtx)
	s.Len(scopes, 2, "removed scope should be absent in the subsequent Get*()")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, goodScope)
	s.NoError(err, "adding a scope with ID and name that used to exist is not an error")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, declarativeScope)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to add imperatively declarative access scope is an error")

	err = s.dataStore.AddAccessScope(s.hasWriteDeclarativeCtx, declarativeScope)
	s.NoError(err, "adding a declarative access scope declaratively is not an error")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, declarativeScope)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to modify imperatively declarative access scope is an error")

	err = s.dataStore.UpdateAccessScope(s.hasWriteDeclarativeCtx, goodScope)
	s.ErrorIs(err, errox.NotAuthorized, "attempting to modify declaratively imperative access scope is an error")

	err = s.dataStore.UpdateAccessScope(s.hasWriteDeclarativeCtx, declarativeScope)
	s.NoError(err, "attempting to modify declaratively declarative access scope is not an error")

	err = s.dataStore.RemoveAccessScope(s.hasWriteCtx, declarativeScope.GetId())
	s.ErrorIs(err, errox.NotAuthorized, "attempting to delete imperatively declarative access scope is an error")

	err = s.dataStore.RemoveAccessScope(s.hasWriteDeclarativeCtx, goodScope.GetId())
	s.ErrorIs(err, errox.NotAuthorized, "attempting to delete declaratively imperative access scope is an error")

	err = s.dataStore.RemoveAccessScope(s.hasWriteDeclarativeCtx, declarativeScope.GetId())
	s.NoError(err, "attempting to delete declaratively declarative access scope is not an error")

	err = s.dataStore.UpsertAccessScope(s.hasWriteCtx, declarativeScope)
	s.ErrorIs(err, errox.NotAuthorized, "upserting imperatively declarative access scope is an error")

	err = s.dataStore.UpsertAccessScope(s.hasWriteDeclarativeCtx, declarativeScope)
	s.NoError(err, "attempting to upsert declaratively declarative access scope is not an error")

	err = s.dataStore.UpsertAccessScope(s.hasWriteDeclarativeCtx, declarativeScope)
	s.NoError(err, "re-upserting declaratively declarative access scope is not an error")

	err = s.dataStore.UpsertAccessScope(s.hasWriteCtx, declarativeScope)
	s.ErrorIs(err, errox.NotAuthorized, "re-upserting imperatively declarative access scope is an error")

	err = s.dataStore.UpsertAccessScope(s.hasWriteDeclarativeCtx, goodScope)
	s.ErrorIs(err, errox.NotAuthorized, "upserting declaratively imperative access scope is an error")

	err = s.dataStore.UpsertAccessScope(s.hasWriteCtx, goodScope)
	s.NoError(err, "attempting to upsert imperatively imperative access scope is not an error")

	err = s.dataStore.UpsertAccessScope(s.hasWriteCtx, goodScope)
	s.NoError(err, "re-upserting imperatively imperative access scope is not an error")

	err = s.dataStore.UpsertAccessScope(s.hasWriteDeclarativeCtx, goodScope)
	s.ErrorIs(err, errox.NotAuthorized, "re-upserting declaratively imperative access scope is an error")

	err = s.dataStore.UpsertAccessScope(s.hasWriteCtx, badScope)
	s.ErrorIs(err, errox.InvalidArgs, "invalid scope for Upsert*() yields an error(imperative resource)")

	err = s.dataStore.UpsertAccessScope(s.hasWriteDeclarativeCtx, badDeclarativeScope)
	s.ErrorIs(err, errox.InvalidArgs, "invalid scope for Upsert*() yields an error(declarative resource)")
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
	s.ErrorIs(err, errox.InvalidArgs, "Cannot create a Role without its PermissionSet and AccessScope existing")

	s.NoError(s.dataStore.AddPermissionSet(s.hasWriteCtx, permissionSet))

	err = s.dataStore.AddRole(s.hasWriteCtx, role)
	s.ErrorIs(err, errox.InvalidArgs, "Cannot create a Role without its AccessScope existing")

	s.NoError(s.dataStore.AddAccessScope(s.hasWriteCtx, scope))
	s.NoError(s.dataStore.AddRole(s.hasWriteCtx, role))

	err = s.dataStore.RemovePermissionSet(s.hasWriteCtx, permissionSet.GetId())
	s.ErrorIs(err, errox.ReferencedByAnotherObject, "cannot delete a PermissionSet referred to by a Role")

	err = s.dataStore.RemoveAccessScope(s.hasWriteCtx, scope.GetId())
	s.ErrorIs(err, errox.ReferencedByAnotherObject, "cannot delete an Access Scope referred to by a Role")

	s.setupGetFilteredReturnValues([]*storage.Group{
		{
			RoleName: role.GetName(),
		},
	}, nil)
	err = s.dataStore.RemoveRole(s.hasWriteCtx, role.GetName())
	s.ErrorIs(err, errox.ReferencedByAnotherObject)

	s.setupGetFilteredReturnValues([]*storage.Group{}, nil)
	err = s.dataStore.RemoveRole(s.hasWriteCtx, role.GetName())
	s.NoError(err)

	s.NoError(s.dataStore.RemovePermissionSet(s.hasWriteCtx, permissionSet.GetId()))
	s.NoError(s.dataStore.RemoveAccessScope(s.hasWriteCtx, scope.GetId()))
}

func (s *roleDataStoreTestSuite) TestGetAndResolveRole() {
	noScopeRole := getValidRole("role without a scope", s.existingPermissionSet.GetId(), "")

	resolvedRole, err := s.dataStore.GetAndResolveRole(s.hasNoneCtx, s.existingRole.GetName())
	s.ErrorIs(err, sac.ErrResourceAccessDenied)
	s.Nil(resolvedRole)

	resolvedRole, err = s.dataStore.GetAndResolveRole(s.hasReadCtx, noScopeRole.GetName())
	s.NoError(err, "no error even if the role does not exist")
	s.Nil(resolvedRole)

	err = s.dataStore.AddRole(s.hasWriteCtx, noScopeRole)
	s.ErrorIs(err, errox.InvalidArgs)

	resolvedRole, err = s.dataStore.GetAndResolveRole(s.hasReadCtx, s.existingRole.GetName())
	s.NoError(err)
	s.Equal(s.existingRole.GetName(), resolvedRole.GetRoleName())
	s.Equal(s.existingPermissionSet.GetResourceToAccess(), resolvedRole.GetPermissions())
	s.Equal(s.existingScope, resolvedRole.GetAccessScope())
}
