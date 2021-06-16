package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	roleStoreMocks "github.com/stackrox/rox/central/role/datastore/internal/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	permissionSetStore "github.com/stackrox/rox/central/role/store/permissionset/rocksdb"
	simpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/rocksdb"
	"github.com/stackrox/rox/central/role/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

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

	dataStore   DataStore
	roleStorage *roleStoreMocks.MockStore
	rocksie     *rocksdb.RocksDB

	existingPermissionSet *storage.PermissionSet
	existingScope         *storage.SimpleAccessScope

	mockCtrl *gomock.Controller
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

	s.mockCtrl = gomock.NewController(s.T())
	s.roleStorage = roleStoreMocks.NewMockStore(s.mockCtrl)

	s.initDataStore(false)
}

func (s *roleDataStoreTestSuite) initDataStore(sacV2Enabled bool) {
	s.rocksie = rocksdbtest.RocksDBForT(s.T())
	permissionSetStorage, err := permissionSetStore.New(s.rocksie)
	s.Require().NoError(err)
	scopeStorage, err := simpleAccessScopeStore.New(s.rocksie)
	s.Require().NoError(err)

	s.dataStore = New(s.roleStorage, permissionSetStorage, scopeStorage, sacV2Enabled)

	// Insert a permission set and an access scope into the test DB.
	s.existingPermissionSet = getValidPermissionSet("permissionset.existing", "existing permissionset")
	s.Require().NoError(permissionSetStorage.Upsert(s.existingPermissionSet))
	s.existingScope = getValidAccessScope("scope.existing", "existing scope")
	s.Require().NoError(scopeStorage.Upsert(s.existingScope))
}

func (s *roleDataStoreTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.rocksie)
	s.mockCtrl.Finish()
}

func (s *roleDataStoreTestSuite) TestEnforcesGet() {
	s.roleStorage.EXPECT().GetRole(gomock.Any()).Times(0)

	role, err := s.dataStore.GetRole(s.hasNoneCtx, "someID")
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(role, "expected return value to be nil")
}

func (s *roleDataStoreTestSuite) TestAllowsGet() {
	s.roleStorage.EXPECT().GetRole(gomock.Any()).Return(nil, nil)

	_, err := s.dataStore.GetRole(s.hasReadCtx, "someID")
	s.NoError(err, "expected no error trying to read with permissions")

	s.roleStorage.EXPECT().GetRole(gomock.Any()).Return(nil, nil)

	_, err = s.dataStore.GetRole(s.hasWriteCtx, "someID")
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *roleDataStoreTestSuite) TestEnforcesGetAll() {
	s.roleStorage.EXPECT().GetAllRoles().Times(0)

	roles, err := s.dataStore.GetAllRoles(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(roles, "expected return value to be nil")
}

func (s *roleDataStoreTestSuite) TestAllowsGetAll() {
	s.roleStorage.EXPECT().GetAllRoles().Return(nil, nil)

	_, err := s.dataStore.GetAllRoles(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.roleStorage.EXPECT().GetAllRoles().Return(nil, nil)

	_, err = s.dataStore.GetAllRoles(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *roleDataStoreTestSuite) TestEnforcesAdd() {
	s.roleStorage.EXPECT().AddRole(gomock.Any()).Times(0)

	err := s.dataStore.AddRole(s.hasNoneCtx, &storage.Role{Name: "role"})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.AddRole(s.hasReadCtx, &storage.Role{Name: "role"})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *roleDataStoreTestSuite) TestAllowsAdd() {
	s.roleStorage.EXPECT().AddRole(gomock.Any()).Return(nil)

	err := s.dataStore.AddRole(s.hasWriteCtx, &storage.Role{Name: "role"})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *roleDataStoreTestSuite) TestEnforcesUpdate() {
	s.roleStorage.EXPECT().AddRole(gomock.Any()).Times(0)

	err := s.dataStore.UpdateRole(s.hasNoneCtx, &storage.Role{Name: "role"})
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.UpdateRole(s.hasReadCtx, &storage.Role{Name: "role"})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *roleDataStoreTestSuite) TestAllowsUpdate() {
	s.roleStorage.EXPECT().UpdateRole(gomock.Any()).Return(nil)

	err := s.dataStore.UpdateRole(s.hasWriteCtx, &storage.Role{Name: "role"})
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *roleDataStoreTestSuite) TestEnforcesRemove() {
	s.roleStorage.EXPECT().RemoveRole(gomock.Any()).Times(0)

	err := s.dataStore.RemoveRole(s.hasNoneCtx, "role")
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.RemoveRole(s.hasReadCtx, "role")
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *roleDataStoreTestSuite) TestAllowsRemove() {
	s.roleStorage.EXPECT().RemoveRole(gomock.Any()).Return(nil)

	err := s.dataStore.RemoveRole(s.hasWriteCtx, "role")
	s.NoError(err, "expected no error trying to write with permissions")
}

////////////////////////////////////////////////////////////////////////////////
// Permission sets                                                            //
//                                                                            //

func getValidPermissionSet(id string, name string) *storage.PermissionSet {
	return &storage.PermissionSet{
		Id:   utils.EnsureValidPermissionSetID(id),
		Name: name,
	}
}

func getInvalidPermissionSet(id string, name string) *storage.PermissionSet {
	return &storage.PermissionSet{
		Id:   utils.EnsureValidPermissionSetID(id),
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
	s.ErrorIs(err, sac.ErrPermissionDenied, "no access for Add*() yields a permission error")

	err = s.dataStore.AddPermissionSet(s.hasReadCtx, goodPermissionSet)
	s.ErrorIs(err, sac.ErrPermissionDenied, "READ access for Add*() yields a permission error")

	err = s.dataStore.AddPermissionSet(s.hasReadCtx, badPermissionSet)
	s.ErrorIs(err, sac.ErrPermissionDenied, "still a permission error for invalid permissionSet")

	err = s.dataStore.UpdatePermissionSet(s.hasNoneCtx, s.existingPermissionSet)
	s.ErrorIs(err, sac.ErrPermissionDenied, "no access for Update*() yields a permission error")

	err = s.dataStore.UpdatePermissionSet(s.hasReadCtx, s.existingPermissionSet)
	s.ErrorIs(err, sac.ErrPermissionDenied, "READ access for Update*() yields a permission error")

	err = s.dataStore.UpdatePermissionSet(s.hasReadCtx, goodPermissionSet)
	s.ErrorIs(err, sac.ErrPermissionDenied, "still a permission error if the object does not exist")

	err = s.dataStore.RemovePermissionSet(s.hasNoneCtx, s.existingPermissionSet.GetId())
	s.ErrorIs(err, sac.ErrPermissionDenied, "no access for Remove*() yields a permission error")

	err = s.dataStore.RemovePermissionSet(s.hasReadCtx, s.existingPermissionSet.GetId())
	s.ErrorIs(err, sac.ErrPermissionDenied, "READ access for Remove*() yields a permission error")

	err = s.dataStore.RemovePermissionSet(s.hasReadCtx, goodPermissionSet.GetId())
	s.ErrorIs(err, sac.ErrPermissionDenied, "still a permission error if the object does not exist")
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

	err := s.dataStore.AddPermissionSet(s.hasWriteCtx, badPermissionSet)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "invalid permission set for Add*() yields an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, clonePermissionSet)
	s.ErrorIs(err, errorhelpers.ErrAlreadyExists, "adding permission set with an existing ID yields an error")

	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, mimicPermissionSet)
	s.ErrorIs(err, errorhelpers.ErrAlreadyExists, "adding permission set with an existing name yields an error")

	err = s.dataStore.UpdatePermissionSet(s.hasWriteCtx, goodPermissionSet)
	s.ErrorIs(err, errorhelpers.ErrNotFound, "updating non-existing permission set yields an error")

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
		Id:   utils.EnsureValidAccessScopeID(id),
		Name: name,
	}
}

func getInvalidAccessScope(id string, name string) *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id:   utils.EnsureValidAccessScopeID(id),
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
	s.ErrorIs(err, sac.ErrPermissionDenied, "no access for Add*() yields a permission error")

	err = s.dataStore.AddAccessScope(s.hasReadCtx, goodScope)
	s.ErrorIs(err, sac.ErrPermissionDenied, "READ access for Add*() yields a permission error")

	err = s.dataStore.AddAccessScope(s.hasReadCtx, badScope)
	s.ErrorIs(err, sac.ErrPermissionDenied, "still a permission error for invalid scope")

	err = s.dataStore.UpdateAccessScope(s.hasNoneCtx, s.existingScope)
	s.ErrorIs(err, sac.ErrPermissionDenied, "no access for Update*() yields a permission error")

	err = s.dataStore.UpdateAccessScope(s.hasReadCtx, s.existingScope)
	s.ErrorIs(err, sac.ErrPermissionDenied, "READ access for Update*() yields a permission error")

	err = s.dataStore.UpdateAccessScope(s.hasReadCtx, goodScope)
	s.ErrorIs(err, sac.ErrPermissionDenied, "still a permission error if the object does not exist")

	err = s.dataStore.RemoveAccessScope(s.hasNoneCtx, s.existingScope.GetId())
	s.ErrorIs(err, sac.ErrPermissionDenied, "no access for Remove*() yields a permission error")

	err = s.dataStore.RemoveAccessScope(s.hasReadCtx, s.existingScope.GetId())
	s.ErrorIs(err, sac.ErrPermissionDenied, "READ access for Remove*() yields a permission error")

	err = s.dataStore.RemoveAccessScope(s.hasReadCtx, goodScope.GetId())
	s.ErrorIs(err, sac.ErrPermissionDenied, "still a permission error if the object does not exist")
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

	err := s.dataStore.AddAccessScope(s.hasWriteCtx, badScope)
	s.ErrorIs(err, errorhelpers.ErrInvalidArgs, "invalid scope for Add*() yields an error")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, cloneScope)
	s.ErrorIs(err, errorhelpers.ErrAlreadyExists, "adding scope with an existing ID yields an error")

	err = s.dataStore.AddAccessScope(s.hasWriteCtx, mimicScope)
	s.ErrorIs(err, errorhelpers.ErrAlreadyExists, "adding scope with an existing name yields an error")

	err = s.dataStore.UpdateAccessScope(s.hasWriteCtx, goodScope)
	s.ErrorIs(err, errorhelpers.ErrNotFound, "updating non-existing scope yields an error")

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

func (s *roleDataStoreTestSuite) TestResolveRoles() {
	permissionSetID := utils.GeneratePermissionSetID()
	permissionSet := getValidPermissionSet(permissionSetID, "meh")
	resourceToAccess := map[string]storage.Access{
		"Image": storage.Access_READ_WRITE_ACCESS,
	}
	legacyRole := &storage.Role{
		Name:             "legacy",
		ResourceToAccess: resourceToAccess,
	}
	regularRole := &storage.Role{
		Name:            "regular",
		PermissionSetId: permissionSetID,
	}

	s.roleStorage.EXPECT().AddRole(gomock.Any()).Times(2)
	err := s.dataStore.AddRole(s.hasWriteCtx, regularRole)
	s.NoError(err)
	err = s.dataStore.AddRole(s.hasWriteCtx, legacyRole)
	s.NoError(err)

	// when SAC is enabled we work with permission sets
	s.initDataStore(true)
	// if there is no permission set with this id, return error
	_, err = s.dataStore.ResolveRoles(s.hasNoneCtx, []*storage.Role{regularRole})
	s.Error(err)

	// if there is permission set with this id, return it in resolved role
	err = s.dataStore.AddPermissionSet(s.hasWriteCtx, permissionSet)
	s.NoError(err)
	resolvedRoles, err := s.dataStore.ResolveRoles(s.hasNoneCtx, []*storage.Role{regularRole})
	s.NoError(err)
	s.Len(resolvedRoles, 1)
	s.Equal(permissionSet, resolvedRoles[0].PermissionSet)

	// when SAC is disabled
	s.initDataStore(false)
	// create fake permission set for legacy role
	resolvedRoles, err = s.dataStore.ResolveRoles(s.hasNoneCtx, []*storage.Role{legacyRole})
	s.NoError(err)
	s.Len(resolvedRoles, 1)
	s.Equal(resourceToAccess, resolvedRoles[0].GetResourceToAccess())
}
