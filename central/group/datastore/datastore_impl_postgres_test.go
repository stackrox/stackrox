//go:build sql_integration

package datastore

import (
	"context"
	"fmt"
	"testing"

	postgresGroupStore "github.com/stackrox/rox/central/group/datastore/internal/store/postgres"
	roleDatastoreMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	authProvidersMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	postgresSchema "github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestGroupsWithPostgres(t *testing.T) {
	suite.Run(t, new(groupsWithPostgresTestSuite))
}

type groupsWithPostgresTestSuite struct {
	suite.Suite

	ctx          context.Context
	testPostgres *pgtest.TestPostgres

	groupsDatastore   DataStore
	mockCtrl          *gomock.Controller
	roleStore         *roleDatastoreMocks.MockDataStore
	authProviderStore *authProvidersMocks.MockStore
}

func (s *groupsWithPostgresTestSuite) SetupSuite() {
	s.mockCtrl = gomock.NewController(s.T())
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	s.testPostgres = pgtest.ForT(s.T())
	s.Require().NotNil(s.testPostgres)

	store := postgresGroupStore.New(s.testPostgres.DB)
	s.roleStore = roleDatastoreMocks.NewMockDataStore(s.mockCtrl)
	s.authProviderStore = authProvidersMocks.NewMockStore(s.mockCtrl)
	s.groupsDatastore = New(store, s.roleStore, s.authProviderStore)
}

func (s *groupsWithPostgresTestSuite) TearDownSuite() {
	s.testPostgres.Teardown(s.T())
}

func (s *groupsWithPostgresTestSuite) TearDownTest() {
	sql := fmt.Sprintf("TRUNCATE %s CASCADE", postgresSchema.GroupsTableName)
	_, err := s.testPostgres.Exec(s.ctx, sql)
	s.NoError(err)
}

func (s *groupsWithPostgresTestSuite) TestAddGroups() {
	group := fixtures.GetGroup()

	s.validRoleAndAuthProvider(group.GetRoleName(), group.GetProps().GetAuthProviderId(), 2)
	// 0. Ensure the group to be added has no ID set.
	group.Props.Id = ""

	// 1. Adding a group should work.
	err := s.groupsDatastore.Add(s.ctx, group)
	s.NoError(err)

	// 2. Adding the _same_ group twice should fail, since (auth provider ID, key, value, role name) should be unique
	group.Props.Id = ""
	err = s.groupsDatastore.Add(s.ctx, group)
	s.Error(err)
	s.ErrorIs(err, errox.AlreadyExists)

	// 3. Adding a different group should work.
	group.RoleName = "headmaster"
	group.Props.Id = ""
	s.validRoleAndAuthProvider(group.GetRoleName(), group.GetProps().GetAuthProviderId(), 1)
	err = s.groupsDatastore.Add(s.ctx, group)
	s.NoError(err)
}

func (s *groupsWithPostgresTestSuite) TestUpdateGroups() {
	group := fixtures.GetGroup()

	s.validRoleAndAuthProvider(group.GetRoleName(), group.GetProps().GetAuthProviderId(), 3)
	// 0. Insert the group.
	group.Props.Id = ""
	err := s.groupsDatastore.Add(s.ctx, group)
	s.NoError(err)

	// 1. Updating the group to be the same shouldn't throw an error as it's a no-op.
	err = s.groupsDatastore.Update(s.ctx, group, false)
	s.NoError(err)

	// 2. Create another group. Updating this group to be equal to the previously added one should fail.
	newGroup := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: "some-authprovider-id",
			Key:            "some-key",
			Value:          "some-value",
		},
		RoleName: "some-role",
	}
	s.validRoleAndAuthProvider(newGroup.GetRoleName(), newGroup.GetProps().GetAuthProviderId(), 1)
	err = s.groupsDatastore.Add(s.ctx, newGroup)
	s.NoError(err)

	newGroup.Props.AuthProviderId = group.GetProps().GetAuthProviderId()
	newGroup.Props.Key = group.GetProps().GetKey()
	newGroup.Props.Value = group.GetProps().GetValue()
	newGroup.RoleName = group.GetRoleName()

	err = s.groupsDatastore.Update(s.ctx, newGroup, false)
	s.Error(err)
	s.ErrorIs(err, errox.AlreadyExists)
}

func (s *groupsWithPostgresTestSuite) TestUpsertGroups() {
	group := fixtures.GetGroup()

	s.validRoleAndAuthProvider(group.GetRoleName(), group.GetProps().GetAuthProviderId(), 3)

	// 0. Insert the group.
	group.Props.Id = ""
	err := s.groupsDatastore.Add(s.ctx, group)
	s.NoError(err)

	// 1. Upserting the same group shouldn't throw an error as it's a no-op.
	err = s.groupsDatastore.Upsert(s.ctx, group)
	s.NoError(err)

	// 2. Upsert another group equal to the previously added one should fail.
	group.Props.Id = ""
	err = s.groupsDatastore.Upsert(s.ctx, group)
	s.Error(err)
	s.ErrorIs(err, errox.AlreadyExists)
}

func (s *groupsWithPostgresTestSuite) TestMutateGroups() {
	group := fixtures.GetGroup()

	s.validRoleAndAuthProvider(group.GetRoleName(), group.GetProps().GetAuthProviderId(), 2)

	// 1. Adding new groups to the store with mutate should work.
	group.Props.Id = ""
	err := s.groupsDatastore.Mutate(s.ctx, nil, nil, []*storage.Group{group}, false)
	s.NoError(err)

	existingGroupID := group.GetProps().GetId()

	// 2. Adding the same group twice should not work.
	group.Props.Id = ""
	err = s.groupsDatastore.Mutate(s.ctx, nil, nil, []*storage.Group{group}, false)
	s.Error(err)
	s.ErrorIs(err, errox.AlreadyExists)

	// 3. Adding another group and updating the existing group to the same values should not work.
	newGroup := &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: "some-authprovider-id",
			Key:            "some-key",
			Value:          "some-value",
		},
		RoleName: "some-role",
	}
	group.RoleName = newGroup.GetRoleName()
	group.Props.AuthProviderId = newGroup.GetProps().GetAuthProviderId()
	group.Props.Key = newGroup.GetProps().GetKey()
	group.Props.Value = newGroup.GetProps().GetValue()
	group.Props.Id = existingGroupID

	s.validRoleAndAuthProvider(group.GetRoleName(), group.GetProps().GetAuthProviderId(), 2)

	err = s.groupsDatastore.Mutate(s.ctx, nil, []*storage.Group{group}, []*storage.Group{newGroup}, false)
	s.Error(err)
	s.ErrorIs(err, errox.AlreadyExists)
}

func (s *groupsWithPostgresTestSuite) validRoleAndAuthProvider(roleName, authProviderID string, times int) {
	mockedRole := &storage.Role{
		Name: roleName,
	}
	mockedAP := &storage.AuthProvider{
		Id: authProviderID,
	}
	s.roleStore.EXPECT().GetRole(gomock.Any(), roleName).Return(mockedRole, true, nil).Times(times)
	s.authProviderStore.EXPECT().GetAuthProvider(gomock.Any(), authProviderID).Return(mockedAP, true, nil).Times(times)
}
