package datastore

import (
	"context"
	"testing"

	storeMocks "github.com/stackrox/rox/central/group/datastore/internal/store/mocks"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	authProvidersMocks "github.com/stackrox/rox/pkg/auth/authproviders/mocks"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestGroupDataStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(groupDataStoreTestSuite))
}

var (
	groupWithID = &storage.Group{Props: &storage.GroupProperties{
		Id:             "123",
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
)

type groupDataStoreTestSuite struct {
	suite.Suite

	hasNoneCtx             context.Context
	hasReadCtx             context.Context
	hasWriteDeclarativeCtx context.Context

	hasWriteCtx context.Context
	dataStore   DataStore

	storage           *storeMocks.MockStore
	mockCtrl          *gomock.Controller
	roleStore         *roleMocks.MockDataStore
	authProviderStore *authProvidersMocks.MockStore
}

func (s *groupDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))
	s.hasWriteDeclarativeCtx = declarativeconfig.WithModifyDeclarativeResource(s.hasWriteCtx)

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.roleStore = roleMocks.NewMockDataStore(s.mockCtrl)
	s.authProviderStore = authProvidersMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage, s.roleStore, s.authProviderStore)
}

func (s *groupDataStoreTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func walkMockFunc(groups []*storage.Group) func(_ context.Context, fn func(group *storage.Group) error) error {
	return func(_ context.Context, fn func(group *storage.Group) error) error {
		for _, g := range groups {
			if err := fn(g); err != nil {
				return err
			}
		}
		return nil
	}
}

func (s *groupDataStoreTestSuite) TestEnforcesGet() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)

	group, err := s.dataStore.Get(s.hasNoneCtx, &storage.GroupProperties{})
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(group, "expected return value to be nil")
}

func (s *groupDataStoreTestSuite) TestAllowsGet() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil)

	_, err := s.dataStore.Get(s.hasReadCtx, &storage.GroupProperties{Id: "1", AuthProviderId: "something"})
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, true, nil).Times(1)

	_, err = s.dataStore.Get(s.hasWriteCtx, &storage.GroupProperties{Id: "1", AuthProviderId: "something"})
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *groupDataStoreTestSuite) TestGet() {
	group := fixtures.GetGroup()
	s.storage.EXPECT().Get(gomock.Any(), group.GetProps().GetId()).Return(group, true, nil).Times(1)

	// Test that can fetch by id
	g, err := s.dataStore.Get(s.hasReadCtx, &storage.GroupProperties{Id: group.GetProps().GetId(),
		AuthProviderId: group.GetProps().GetAuthProviderId()})
	s.NoError(err)
	s.Equal(group, g)

	// Test that a non-existing group will yield errox.NotFound.
	s.storage.EXPECT().Get(gomock.Any(), group.GetProps().GetId()).Return(nil, false, nil).Times(1)
	g, err = s.dataStore.Get(s.hasReadCtx, group.GetProps())
	s.Nil(g)
	s.ErrorIs(err, errox.NotFound)
}

func (s *groupDataStoreTestSuite) TestGetWithoutID() {
	group := fixtures.GetGroup()
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)

	g, err := s.dataStore.Get(s.hasReadCtx, &storage.GroupProperties{
		Id:             "",
		Traits:         nil,
		AuthProviderId: group.GetProps().GetAuthProviderId(),
		Key:            group.GetProps().GetKey(),
		Value:          group.GetProps().GetValue(),
	})
	s.Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(g)
}

func (s *groupDataStoreTestSuite) TestEnforcesGetAll() {
	s.storage.EXPECT().GetAll(gomock.Any()).Times(0)

	groups, err := s.dataStore.GetAll(s.hasNoneCtx)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(groups, "expected return value to be nil")
}

func (s *groupDataStoreTestSuite) TestAllowsGetAll() {
	s.storage.EXPECT().GetAll(gomock.Any()).Return(nil, nil)

	_, err := s.dataStore.GetAll(s.hasReadCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().GetAll(gomock.Any()).Return(nil, nil).Times(1)

	_, err = s.dataStore.GetAll(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesGetFiltered() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(0)

	groups, err := s.dataStore.GetFiltered(s.hasNoneCtx, func(_ *storage.Group) bool { return true })
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(groups, "expected return value to be nil")
}

func (s *groupDataStoreTestSuite) TestAllowsGetFiltered() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil)

	_, err := s.dataStore.GetFiltered(s.hasReadCtx, func(_ *storage.Group) bool { return true })
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err = s.dataStore.GetFiltered(s.hasWriteCtx, func(_ *storage.Group) bool { return true })
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *groupDataStoreTestSuite) TestGetFiltered() {
	groups := fixtures.GetGroups()
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(2).DoAndReturn(walkMockFunc(groups))

	actualGroups, err := s.dataStore.GetFiltered(s.hasWriteCtx, func(*storage.Group) bool { return false })
	s.NoError(err)
	s.Empty(actualGroups)

	// Test with a selective filter
	actualGroups, err = s.dataStore.GetFiltered(s.hasWriteCtx, func(group *storage.Group) bool {
		return group.GetProps().GetAuthProviderId() == "authProvider1" || group.GetProps().GetKey() == "Attribute2"
	})
	expectedGroups := []*storage.Group{
		groups[1], groups[2], groups[3], groups[4], groups[6],
	}
	s.NoError(err)
	s.ElementsMatch(expectedGroups, actualGroups)
}

func (s *groupDataStoreTestSuite) TestEnforcesWalk() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(0)

	groups, err := s.dataStore.Walk(s.hasNoneCtx, "provider", nil)
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(groups, "expected return value to be nil")
}

func (s *groupDataStoreTestSuite) TestAllowsWalk() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil)

	_, err := s.dataStore.Walk(s.hasReadCtx, "provider", nil)
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	_, err = s.dataStore.Walk(s.hasWriteCtx, "provider", nil)
	s.NoError(err, "expected no error trying to read with permissions")
}

func (s *groupDataStoreTestSuite) TestWalk() {
	groups := fixtures.GetGroups()
	expectedGroups := []*storage.Group{
		groups[1],
		groups[2],
		groups[3],
	}

	attributes := map[string][]string{
		"Attribute1": {
			"Value1",
		},
		"Attribute2": {
			"Value2",
		},
	}

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc(groups))

	actualGroups, err := s.dataStore.Walk(s.hasWriteCtx, "authProvider1", attributes)
	s.NoError(err)
	s.ElementsMatch(expectedGroups, actualGroups)
}

func (s *groupDataStoreTestSuite) TestEnforcesAdd() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	grp := &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Add(s.hasNoneCtx, grp)
	s.Error(err, "expected an error trying to write without permissions")

	grp = &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err = s.dataStore.Add(s.hasReadCtx, grp)
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsAdd() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.validRoleAndAuthProvider("123", "123", storage.Traits_IMPERATIVE, 1)

	grp := &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Add(s.hasWriteCtx, grp)
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	grp := &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Update(s.hasNoneCtx, grp, false)
	s.Error(err, "expected an error trying to write without permissions")

	grp = &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err = s.dataStore.Update(s.hasReadCtx, grp, false)
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsUpdate() {
	s.expectGet(1, fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE))
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.validRoleAndAuthProvider("123", "123", storage.Traits_IMPERATIVE, 1)

	grp := &storage.Group{Props: &storage.GroupProperties{
		Id:             "1",
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Update(s.hasWriteCtx, grp, false)
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesMutate() {
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Times(0)
	s.storage.EXPECT().DeleteMany(gomock.Any(), gomock.Any()).Times(0)

	grp := &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Mutate(s.hasNoneCtx, []*storage.Group{groupWithID}, []*storage.Group{groupWithID},
		[]*storage.Group{grp}, false)
	s.Error(err, "expected an error trying to write without permissions")

	grp = &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err = s.dataStore.Mutate(s.hasReadCtx, []*storage.Group{groupWithID}, []*storage.Group{groupWithID},
		[]*storage.Group{grp}, false)
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsMutate() {
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil).Times(2) // two calls * two operations (add, update)
	s.storage.EXPECT().DeleteMany(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE), true, nil).Times(2)
	s.validRoleAndAuthProvider("123", "123", storage.Traits_IMPERATIVE, 2)

	grp := &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{groupWithID}, []*storage.Group{groupWithID},
		[]*storage.Group{grp}, false)
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *groupDataStoreTestSuite) TestMutate() {
	toRemove := fixtures.GetGroups()[6]
	toUpdate := fixtures.GetGroups()[5]
	toAdd := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "DifferentAttribute",
				Value:          "IsNotCaptain",
			},
			RoleName: "notcaptain",
		},
	}
	s.validRoleAndAuthProvider(toUpdate.GetRoleName(), toUpdate.GetProps().GetAuthProviderId(), storage.Traits_IMPERATIVE, 1)
	s.validRoleAndAuthProvider(toAdd[0].GetRoleName(), toAdd[0].GetProps().GetAuthProviderId(), storage.Traits_DECLARATIVE, 1)
	s.expectGet(2, fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE))
	gomock.InOrder(
		s.storage.EXPECT().UpsertMany(gomock.Any(), toAdd),
		s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{toUpdate}),
		s.storage.EXPECT().DeleteMany(gomock.Any(), []string{toRemove.GetProps().GetId()}),
	)

	s.NoError(s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{toRemove}, []*storage.Group{toUpdate}, toAdd, false))
}

func (s *groupDataStoreTestSuite) TestCannotAddDefaultGroupIfOneAlreadyExists() {
	defaultGroup := &storage.Group{
		RoleName: "admin",
		Props: &storage.GroupProperties{
			AuthProviderId: "defaultGroup1",
			Id:             "some-id",
		},
	}
	initialGroup := &storage.Group{
		RoleName: "Manager",
		Props: &storage.GroupProperties{
			AuthProviderId: "defaultGroup1",
			Key:            "something",
			Value:          "someone",
			Id:             "some-id-3",
		},
	}

	cases := []struct {
		name           string
		existingGroups []*storage.Group
		groupToAdd     *storage.Group
		shouldError    bool
	}{
		{
			"No error when setting up a non-default group when no default exists",
			[]*storage.Group{},
			initialGroup.Clone(),
			false,
		},
		{
			"No error when setting up a default group when no default exists",
			[]*storage.Group{},
			defaultGroup.Clone(),
			false,
		},
		{
			"No error when setting up a non-default group when a default already exists",
			[]*storage.Group{defaultGroup},
			initialGroup.Clone(),
			false,
		},
		{
			"Error when setting up a default group when a default already exists",
			[]*storage.Group{defaultGroup},
			defaultGroup.Clone(),
			true,
		},
	}

	for _, c := range cases {
		s.T().Run(c.name, func(t *testing.T) {

			c.groupToAdd.GetProps().Id = "" // clear it out so that the data store doesn't error out

			// If default group, then expect call to Walk (to find if there are other default groups)
			if c.groupToAdd.GetProps().GetKey() == "" && c.groupToAdd.GetProps().GetValue() == "" {
				s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc(c.existingGroups)).Times(2)
			} else {
				// Otherwise, no call to Walk will be made
				s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(0)
			}

			if c.shouldError {
				// Validate Add returns an error if duplicate default group
				s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(0)
				err := s.dataStore.Add(s.hasWriteCtx, c.groupToAdd.Clone())
				s.Error(err)
				s.ErrorIs(err, errox.AlreadyExists)

				// Validate Mutate with additions returns an error if duplicate default group
				s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil).Times(0)
				err = s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{}, []*storage.Group{}, []*storage.Group{c.groupToAdd.Clone()}, false)
				s.Error(err)
				s.ErrorIs(err, errox.AlreadyExists)
			} else {
				s.validRoleAndAuthProvider(c.groupToAdd.GetRoleName(), c.groupToAdd.GetProps().GetAuthProviderId(), storage.Traits_IMPERATIVE, 2)
				// Validate Add doesn't error if it's a new default
				s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				s.NoError(s.dataStore.Add(s.hasWriteCtx, c.groupToAdd.Clone()))

				// Validate  Mutate with additions doesn't error if it's a new default
				s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				s.NoError(s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{}, []*storage.Group{}, []*storage.Group{c.groupToAdd.Clone()}, false))
			}
		})
	}
}

func (s *groupDataStoreTestSuite) TestUpdateToDefaultGroupIfOneAlreadyExists() {
	defaultGroup := &storage.Group{
		RoleName: "admin",
		Props: &storage.GroupProperties{
			AuthProviderId: "defaultGroup1",
			Id:             "some-id",
		},
	}
	initialGroup := &storage.Group{
		RoleName: "Manager",
		Props: &storage.GroupProperties{
			AuthProviderId: "defaultGroup1",
			Key:            "something",
			Value:          "someone",
			Id:             "some-id-3",
		},
	}
	s.expectGet(2, fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE))
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc([]*storage.Group{initialGroup, defaultGroup})).Times(2)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)     // No update should happen
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Times(0) // No updates should happen

	// Unset Key / Value fields, making it a default group.
	updatedGroup := initialGroup.Clone()
	updatedGroup.GetProps().Key = ""
	updatedGroup.GetProps().Value = ""

	// Ensure a "AlreadyExists" error is yielded when trying to update the group.
	err := s.dataStore.Update(s.hasWriteCtx, updatedGroup.Clone(), false)
	s.Error(err)
	s.ErrorIs(err, errox.AlreadyExists)

	// Ensure a "AlreadyExists" error is yielded when trying to update the group using Mutate.
	err = s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{}, []*storage.Group{updatedGroup.Clone()}, []*storage.Group{}, false)
	s.Error(err)
	s.ErrorIs(err, errox.AlreadyExists)
}

func (s *groupDataStoreTestSuite) TestCanUpdateExistingDefaultGroup() {
	defaultGroup := &storage.Group{
		RoleName: "admin",
		Props: &storage.GroupProperties{
			AuthProviderId: "defaultGroup1",
			Id:             "some-id",
		},
	}
	initialGroup := &storage.Group{
		RoleName: "Manager",
		Props: &storage.GroupProperties{
			AuthProviderId: "defaultGroup1",
			Key:            "something",
			Value:          "someone",
			Id:             "some-id-3",
		},
	}
	s.validRoleAndAuthProvider("admin", "defaultGroup1", storage.Traits_IMPERATIVE, 2)
	s.validRoleAndAuthProvider("non-admin", "defaultGroup1", storage.Traits_IMPERATIVE, 4)

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc([]*storage.Group{initialGroup, defaultGroup})).AnyTimes()

	// 1. Updating the default group's role should work.
	defaultGroup.RoleName = "non-admin" // Using the same defaultGroup object so that the Walk closure is also updated correctly

	s.storage.EXPECT().Upsert(gomock.Any(), defaultGroup)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{defaultGroup})

	s.expectGet(2, fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE))
	s.NoError(s.dataStore.Update(s.hasWriteCtx, defaultGroup, false))
	s.NoError(s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{}, []*storage.Group{defaultGroup}, []*storage.Group{}, false))

	// 2. Update the default group to a non-default group.
	defaultGroup.GetProps().Key = "email" // Update the properties to make it a non-default group.
	defaultGroup.GetProps().Value = "test@example.com"

	s.storage.EXPECT().Upsert(gomock.Any(), defaultGroup)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{defaultGroup})

	s.expectGet(2, fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE))
	s.NoError(s.dataStore.Update(s.hasWriteCtx, defaultGroup, false))
	s.NoError(s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{}, []*storage.Group{defaultGroup}, []*storage.Group{}, false))

	// 3. Adding another default group back in should now work, as we have made the existing default group a non-default group.
	newDefaultGroup := &storage.Group{
		RoleName: "admin",
		Props: &storage.GroupProperties{
			AuthProviderId: "defaultGroup1",
		},
	}

	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, g *storage.Group) {
		g.GetProps().Id = ""
		s.Equal(newDefaultGroup, g)
	})
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Do(func(ctx context.Context, groups []*storage.Group) {
		for _, g := range groups {
			g.GetProps().Id = ""
			s.Equal(newDefaultGroup, g)
		}
	})

	s.NoError(s.dataStore.Add(s.hasWriteCtx, newDefaultGroup.Clone()))
	s.NoError(s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{}, []*storage.Group{}, []*storage.Group{newDefaultGroup}, false))
}

func (s *groupDataStoreTestSuite) TestEnforcesRemove() {
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.Remove(s.hasNoneCtx, groupWithID.GetProps(), false)
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.Remove(s.hasReadCtx, groupWithID.GetProps(), false)
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsRemove() {
	s.expectGet(1, fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE))
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.Remove(s.hasWriteCtx, groupWithID.GetProps(), false)
	s.NoError(err, "expected no error trying to write with permissions")
}

func (s *groupDataStoreTestSuite) expectGet(times int, group *storage.Group) *gomock.Call {
	return s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(group, true, nil).Times(times)
}

func (s *groupDataStoreTestSuite) TestValidateGroup() {
	invalidGroups := []*storage.Group{
		{}, // empty props
		{
			Props: &storage.GroupProperties{}, // No auth provider id
		},
		{
			Props: &storage.GroupProperties{ // Value without key
				AuthProviderId: "abcd",
				Value:          "val-1",
			},
		},
		{
			Props: &storage.GroupProperties{ // No role
				AuthProviderId: "abcd",
			},
		},
	}

	for _, g := range invalidGroups {
		err := s.dataStore.Add(s.hasWriteCtx, g)
		s.Error(err)
		s.ErrorIs(err, errox.InvalidArgs)

		err = s.dataStore.Update(s.hasWriteCtx, g, false)
		s.Error(err)
		s.ErrorIs(err, errox.InvalidArgs)

		err = s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{g}, nil, nil, false)
		s.Error(err)
		s.ErrorIs(err, errox.InvalidArgs)

		err = s.dataStore.Mutate(s.hasWriteCtx, nil, []*storage.Group{g}, nil, false)
		s.Error(err)
		s.ErrorIs(err, errox.InvalidArgs)

		err = s.dataStore.Mutate(s.hasWriteCtx, nil, nil, []*storage.Group{g}, false)
		s.Error(err)
		s.ErrorIs(err, errox.InvalidArgs)
	}
}

func (s *groupDataStoreTestSuite) TestUpdateMutableToImmutable() {
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(&storage.Group{
		Props: &storage.GroupProperties{
			Id:             "id",
			AuthProviderId: "apid",
			Traits: &storage.Traits{
				MutabilityMode: storage.Traits_ALLOW_MUTATE,
			},
		},
		RoleName: "Admin",
	}, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.validRoleAndAuthProvider("Admin", "apid", storage.Traits_IMPERATIVE, 1)

	err := s.dataStore.Update(s.hasWriteCtx, &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "id",
			AuthProviderId: "apid",
			Traits: &storage.Traits{
				MutabilityMode: storage.Traits_ALLOW_MUTATE_FORCED,
			},
			Key:   "abc",
			Value: "dfg",
		},
		RoleName: "Admin",
	}, false)
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestUpdateImmutableNoForce() {
	expectedGroup := fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE_FORCED)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedGroup, true, nil).Times(1)

	updatedGroup := expectedGroup.Clone()
	updatedGroup.GetProps().Key = "something"
	updatedGroup.GetProps().Value = "else"

	err := s.dataStore.Update(s.hasWriteCtx, updatedGroup, false)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *groupDataStoreTestSuite) TestUpdateImmutableForce() {
	expectedGroup := fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE_FORCED)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedGroup, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.validRoleAndAuthProvider(expectedGroup.GetRoleName(), expectedGroup.GetProps().GetAuthProviderId(), storage.Traits_IMPERATIVE, 1)

	updatedGroup := expectedGroup.Clone()
	updatedGroup.GetProps().Key = "something"
	updatedGroup.GetProps().Value = "else"

	err := s.dataStore.Update(s.hasWriteCtx, updatedGroup, true)
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestDeleteImmutableNoForce() {
	expectedGroup := fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE_FORCED)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedGroup, true, nil).Times(1)

	err := s.dataStore.Remove(s.hasWriteCtx, expectedGroup.GetProps(), false)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *groupDataStoreTestSuite) TestDeleteImmutableForce() {
	expectedGroup := fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE_FORCED)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedGroup, true, nil).Times(1)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.Remove(s.hasWriteCtx, expectedGroup.GetProps(), true)
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestDefaultGroupCannotBeImmutable() {
	group := &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "id",
			AuthProviderId: "apid",
			Traits: &storage.Traits{
				MutabilityMode: storage.Traits_ALLOW_MUTATE_FORCED,
			},
		},
	}
	err := s.dataStore.Update(s.hasWriteCtx, group, false)
	s.ErrorIs(err, errox.InvalidArgs)
	err = s.dataStore.Add(s.hasWriteCtx, group)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *groupDataStoreTestSuite) TestMutateGroupNoForce() {
	mutableGroup := fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE)
	immutableGroup := fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE_FORCED)

	s.validRoleAndAuthProvider(mutableGroup.GetRoleName(), mutableGroup.GetProps().GetAuthProviderId(), storage.Traits_IMPERATIVE, 1)
	// 1. Try and remove an immutable group via mutate without force. This should fail.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mutableGroup, true, nil),
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(immutableGroup, true, nil),
	)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{mutableGroup}).Return(nil)
	err := s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{immutableGroup}, []*storage.Group{mutableGroup}, nil, false)
	s.Error(err)
	s.ErrorIs(err, errox.InvalidArgs)

	// 2. Try and update an immutable group via mutate without force. This should fail.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(immutableGroup, true, nil),
	)

	err = s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{mutableGroup}, []*storage.Group{immutableGroup}, nil, false)
	s.Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *groupDataStoreTestSuite) TestMutateGroupForce() {
	mutableGroup := fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE)
	immutableGroup := fixtures.GetGroupWithMutability(storage.Traits_ALLOW_MUTATE_FORCED)

	s.validRoleAndAuthProvider(mutableGroup.GetRoleName(), mutableGroup.GetProps().GetAuthProviderId(), storage.Traits_IMPERATIVE, 2)

	// 1. Try and remove an immutable group via mutate with force.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mutableGroup, true, nil),
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(immutableGroup, true, nil),
	)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{mutableGroup}).Return(nil).Times(1)
	s.storage.EXPECT().DeleteMany(gomock.Any(), []string{immutableGroup.GetProps().GetId()}).Return(nil).Times(1)
	err := s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{immutableGroup}, []*storage.Group{mutableGroup}, nil, true)
	s.NoError(err)

	// 2. Try and update an immutable group via mutate with force.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(immutableGroup, true, nil),
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mutableGroup, true, nil),
	)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{immutableGroup}).Return(nil).Times(1)
	s.storage.EXPECT().DeleteMany(gomock.Any(), []string{mutableGroup.GetProps().GetId()}).Return(nil).Times(1)
	err = s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{mutableGroup}, []*storage.Group{immutableGroup}, nil, true)
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestRemoveAllWithEmptyProperties() {
	// 1. Try and remove groups without properties without running into any issues.
	groupsWithoutProperties := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				Id: "id1",
			},
			RoleName: "i don't",
		},

		{
			Props: &storage.GroupProperties{
				Id: "id2",
			},
			RoleName: "know anything",
		},
	}
	gomock.InOrder(
		s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc(groupsWithoutProperties)),
		s.storage.EXPECT().Delete(gomock.Any(), groupsWithoutProperties[0].GetProps().GetId()).Return(nil),
		s.storage.EXPECT().Delete(gomock.Any(), groupsWithoutProperties[1].GetProps().GetId()).Return(nil),
	)

	err := s.dataStore.RemoveAllWithEmptyProperties(s.hasWriteCtx)
	s.NoError(err)

	// 2. Try and remove groups without properties with some groups not having an ID.
	groupsWithoutProperties = []*storage.Group{
		{
			Props: &storage.GroupProperties{
				Id: "id1",
			},
			RoleName: "i don't",
		},
		{
			Props:    &storage.GroupProperties{},
			RoleName: "this is",
		},
		{
			Props: &storage.GroupProperties{
				Id: "id2",
			},
			RoleName: "know anything",
		},
	}
	gomock.InOrder(
		s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc(groupsWithoutProperties)),
		s.storage.EXPECT().Delete(gomock.Any(), groupsWithoutProperties[0].GetProps().GetId()).Return(nil),
		s.storage.EXPECT().Delete(gomock.Any(), groupsWithoutProperties[2].GetProps().GetId()).Return(nil),
	)

	err = s.dataStore.RemoveAllWithEmptyProperties(s.hasWriteCtx)
	s.Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *groupDataStoreTestSuite) TestUpdateDeclarativeViaAPI() {
	expectedGroup := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedGroup, true, nil).Times(1)

	updatedGroup := expectedGroup.Clone()
	updatedGroup.GetProps().Key = "something"
	updatedGroup.GetProps().Value = "else"

	err := s.dataStore.Update(s.hasWriteCtx, updatedGroup, false)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *groupDataStoreTestSuite) TestUpdateDeclarativeViaConfig() {
	expectedGroup := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedGroup, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.validRoleAndAuthProvider(expectedGroup.GetRoleName(), expectedGroup.GetProps().GetAuthProviderId(), storage.Traits_DECLARATIVE, 1)

	updatedGroup := expectedGroup.Clone()
	updatedGroup.GetProps().Key = "something"
	updatedGroup.GetProps().Value = "else"

	err := s.dataStore.Update(s.hasWriteDeclarativeCtx, updatedGroup, true)
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestDeleteDeclarativeViaAPI() {
	expectedGroup := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedGroup, true, nil).Times(1)

	err := s.dataStore.Remove(s.hasWriteCtx, expectedGroup.GetProps(), false)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *groupDataStoreTestSuite) TestDeleteDeclarativeViaConfig() {
	expectedGroup := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedGroup, true, nil).Times(1)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.Remove(s.hasWriteDeclarativeCtx, expectedGroup.GetProps(), true)
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestMutateGroupViaAPI() {
	imperativeGroup := fixtures.GetGroupWithOrigin(storage.Traits_IMPERATIVE)
	declarativeGroup := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)
	declarativeGroup.RoleName = "test-role-2"
	declarativeGroup.Props.AuthProviderId = "authProviderId2"

	s.validRoleAndAuthProvider(imperativeGroup.GetRoleName(), imperativeGroup.GetProps().GetAuthProviderId(), storage.Traits_IMPERATIVE, 1)

	// 1. Try and remove a declarative group via API. This should fail.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(imperativeGroup, true, nil),
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(declarativeGroup, true, nil),
	)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{imperativeGroup}).Return(nil)
	err := s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{declarativeGroup}, []*storage.Group{imperativeGroup}, nil, false)
	s.Error(err)
	s.ErrorIs(err, errox.NotAuthorized)

	// 2. Try and update a declarative group via API. This should fail.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(declarativeGroup, true, nil),
	)

	err = s.dataStore.Mutate(s.hasWriteCtx, []*storage.Group{imperativeGroup}, []*storage.Group{declarativeGroup}, nil, false)
	s.Error(err)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *groupDataStoreTestSuite) TestMutateGroupViaConfig() {
	imperativeGroup := fixtures.GetGroupWithOrigin(storage.Traits_IMPERATIVE)
	declarativeGroup := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)
	declarativeGroup.RoleName = "test-role-2"
	declarativeGroup.Props.AuthProviderId = "authProviderId2"

	s.validRoleAndAuthProvider(declarativeGroup.GetRoleName(), declarativeGroup.GetProps().GetAuthProviderId(), storage.Traits_DECLARATIVE, 2)

	// 1. Try mutate(remove declarative, update imperative) groups via config. This should fail.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(imperativeGroup, true, nil),
	)
	err := s.dataStore.Mutate(s.hasWriteDeclarativeCtx, []*storage.Group{declarativeGroup}, []*storage.Group{imperativeGroup}, nil, true)
	s.Error(err)
	s.ErrorIs(err, errox.NotAuthorized)

	// 2. Try mutate(update declarative, remove imperative) groups via config. This should fail.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(declarativeGroup, true, nil),
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(imperativeGroup, true, nil),
	)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{declarativeGroup}).Return(nil).Times(1)
	err = s.dataStore.Mutate(s.hasWriteDeclarativeCtx, []*storage.Group{imperativeGroup}, []*storage.Group{declarativeGroup}, nil, true)
	s.Error(err)
	s.ErrorIs(err, errox.NotAuthorized)

	// 3. Try update declarative group via config.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(declarativeGroup, true, nil),
	)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{declarativeGroup}).Return(nil).Times(1)
	err = s.dataStore.Mutate(s.hasWriteDeclarativeCtx, nil, []*storage.Group{declarativeGroup}, nil, true)
	s.NoError(err)

	// 4. Try delete declarative group via config.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(declarativeGroup, true, nil),
	)
	s.storage.EXPECT().DeleteMany(gomock.Any(), []string{declarativeGroup.GetProps().GetId()}).Return(nil).Times(1)
	err = s.dataStore.Mutate(s.hasWriteDeclarativeCtx, []*storage.Group{declarativeGroup}, nil, nil, true)
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestUpsertImperativeViaConfig() {
	group := fixtures.GetGroupWithOrigin(storage.Traits_IMPERATIVE)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).Times(1)

	err := s.dataStore.Upsert(s.hasWriteDeclarativeCtx, group)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *groupDataStoreTestSuite) TestUpsertImperativeViaAPI() {
	group := fixtures.GetGroupWithOrigin(storage.Traits_IMPERATIVE)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.validRoleAndAuthProvider(group.GetRoleName(), group.GetProps().GetAuthProviderId(), storage.Traits_IMPERATIVE, 1)

	err := s.dataStore.Upsert(s.hasWriteCtx, group)
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestUpsertDeclarativeViaAPI() {
	group := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).Times(1)

	err := s.dataStore.Upsert(s.hasWriteCtx, group)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *groupDataStoreTestSuite) TestUpsertDeclarativeViaConfig() {
	group := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	s.validRoleAndAuthProvider(group.GetRoleName(), group.GetProps().GetAuthProviderId(), storage.Traits_DECLARATIVE, 1)

	err := s.dataStore.Upsert(s.hasWriteDeclarativeCtx, group)
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestUpsertChangeDeclarativeOrigin() {
	existingGroup := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(existingGroup, true, nil).Times(1)

	updatedGroup := existingGroup.Clone()
	updatedGroup.GetProps().Key = "something"
	updatedGroup.GetProps().Value = "else"
	updatedGroup.GetProps().Traits = &storage.Traits{
		Origin: storage.Traits_IMPERATIVE,
	}

	err := s.dataStore.Upsert(s.hasWriteDeclarativeCtx, updatedGroup)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *groupDataStoreTestSuite) TestUpsertChangeImperativeOrigin() {
	existingGroup := fixtures.GetGroupWithOrigin(storage.Traits_IMPERATIVE)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(existingGroup, true, nil).Times(1)

	updatedGroup := existingGroup.Clone()
	updatedGroup.GetProps().Key = "something"
	updatedGroup.GetProps().Value = "else"
	updatedGroup.GetProps().Traits = &storage.Traits{
		Origin: storage.Traits_DECLARATIVE,
	}

	err := s.dataStore.Upsert(s.hasWriteCtx, updatedGroup)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *groupDataStoreTestSuite) TestAddImperativeViaConfig() {
	group := fixtures.GetGroupWithOrigin(storage.Traits_IMPERATIVE)
	group.Props.Id = ""

	err := s.dataStore.Add(s.hasWriteDeclarativeCtx, group)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *groupDataStoreTestSuite) TestAddDeclarativeViaAPI() {
	group := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)
	group.Props.Id = ""

	err := s.dataStore.Add(s.hasWriteCtx, group)
	s.ErrorIs(err, errox.NotAuthorized)
}

func (s *groupDataStoreTestSuite) TestAddDeclarativeViaConfig() {
	group := fixtures.GetGroupWithOrigin(storage.Traits_DECLARATIVE)
	group.Props.Id = ""

	s.validRoleAndAuthProvider(group.GetRoleName(), group.GetProps().GetAuthProviderId(), storage.Traits_DECLARATIVE, 1)

	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.Add(s.hasWriteDeclarativeCtx, group)
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) validRoleAndAuthProvider(roleName, authProviderID string, origin storage.Traits_Origin, times int) {
	mockedRole := &storage.Role{
		Name: roleName,
		Traits: &storage.Traits{
			Origin: origin,
		},
	}
	mockedAP := &storage.AuthProvider{
		Id: authProviderID,
		Traits: &storage.Traits{
			Origin: origin,
		},
	}
	s.roleStore.EXPECT().GetRole(gomock.Any(), roleName).Return(mockedRole, true, nil).Times(times)
	s.authProviderStore.EXPECT().GetAuthProvider(gomock.Any(), authProviderID).Return(mockedAP, true, nil).Times(times)
}
