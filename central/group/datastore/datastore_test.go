package datastore

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	storeMocks "github.com/stackrox/rox/central/group/datastore/internal/store/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
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

	hasNoneCtx        context.Context
	hasReadCtx        context.Context
	hasWriteCtx       context.Context
	hasWriteAccessCtx context.Context

	dataStore DataStore
	storage   *storeMocks.MockStore

	mockCtrl *gomock.Controller
}

func (s *groupDataStoreTestSuite) SetupTest() {
	s.hasNoneCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.DenyAllAccessScopeChecker())
	s.hasReadCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Group)))
	s.hasWriteCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Group)))
	s.hasWriteAccessCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access)))

	s.mockCtrl = gomock.NewController(s.T())
	s.storage = storeMocks.NewMockStore(s.mockCtrl)
	s.dataStore = New(s.storage)
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
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil)

	_, err := s.dataStore.Get(s.hasReadCtx, &storage.GroupProperties{Id: "1"})
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil).Times(2)

	_, err = s.dataStore.Get(s.hasWriteCtx, &storage.GroupProperties{Id: "1"})
	s.NoError(err, "expected no error trying to read with permissions")

	_, err = s.dataStore.Get(s.hasWriteAccessCtx, &storage.GroupProperties{Id: "1"})
	s.NoError(err, "expected no error trying to read with Access permissions")
}

func (s *groupDataStoreTestSuite) TestGet() {
	group := fixtures.GetGroup()
	s.storage.EXPECT().Get(gomock.Any(), group.GetProps().GetId()).Return(group, true, nil)

	// Test that can fetch by id
	g, err := s.dataStore.Get(s.hasReadCtx, &storage.GroupProperties{Id: group.GetProps().GetId()})
	s.NoError(err)
	s.Equal(group, g)
}

// TODO(ROX-11592): This can be removed once retrieving the group by its properties is fully deprecated.
func (s *groupDataStoreTestSuite) TestGetFetchesByProps() {
	groups := fixtures.GetGroups()
	expectedGroup := groups[6]
	expectedProps := expectedGroup.GetProps()
	expectedProps.Id = "" // clear out the id for the purposes of this test
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc(groups))

	// Test that can fetch by props
	g, err := s.dataStore.Get(s.hasReadCtx, expectedGroup.GetProps())
	s.NoError(err)
	s.Equal(expectedGroup, g)
}

// TODO(ROX-11592): This can be removed once retrieving the group by its properties is fully deprecated.
func (s *groupDataStoreTestSuite) TestGetByPropsErrorsIfAmbiguous() {
	storedGroups := []*storage.Group{
		fixtures.GetGroups()[6],
		fixtures.GetGroups()[6],
	}
	// Clear out the id for each groups to make them all share the same props
	storedGroups[0].GetProps().Id = ""
	storedGroups[1].GetProps().Id = ""
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc(storedGroups))

	// Test that fetching by props when ambiguous will result in error
	g, err := s.dataStore.Get(s.hasReadCtx, storedGroups[0].GetProps())
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

	s.storage.EXPECT().GetAll(gomock.Any()).Return(nil, nil).Times(2)

	_, err = s.dataStore.GetAll(s.hasWriteCtx)
	s.NoError(err, "expected no error trying to read with permissions")

	_, err = s.dataStore.GetAll(s.hasWriteAccessCtx)
	s.NoError(err, "expected no error trying to read with Access permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesGetFiltered() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(0)

	groups, err := s.dataStore.GetFiltered(s.hasNoneCtx, func(_ *storage.GroupProperties) bool { return true })
	s.NoError(err, "expected no error, should return nil without access")
	s.Nil(groups, "expected return value to be nil")
}

func (s *groupDataStoreTestSuite) TestAllowsGetFiltered() {
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil)

	_, err := s.dataStore.GetFiltered(s.hasReadCtx, func(_ *storage.GroupProperties) bool { return true })
	s.NoError(err, "expected no error trying to read with permissions")

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	_, err = s.dataStore.GetFiltered(s.hasWriteCtx, func(_ *storage.GroupProperties) bool { return true })
	s.NoError(err, "expected no error trying to read with permissions")

	_, err = s.dataStore.GetFiltered(s.hasWriteAccessCtx, func(_ *storage.GroupProperties) bool { return true })
	s.NoError(err, "expected no error trying to read with Access permissions")
}

func (s *groupDataStoreTestSuite) TestGetFiltered() {
	groups := fixtures.GetGroups()
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(2).DoAndReturn(walkMockFunc(groups))

	actualGroups, err := s.dataStore.GetFiltered(s.hasWriteAccessCtx, func(*storage.GroupProperties) bool { return false })
	s.NoError(err)
	s.Empty(actualGroups)

	// Test with a selective filter
	actualGroups, err = s.dataStore.GetFiltered(s.hasWriteAccessCtx, func(props *storage.GroupProperties) bool {
		return props.GetAuthProviderId() == "authProvider1" || props.GetKey() == "Attribute2"
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

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	_, err = s.dataStore.Walk(s.hasWriteCtx, "provider", nil)
	s.NoError(err, "expected no error trying to read with permissions")

	_, err = s.dataStore.Walk(s.hasWriteAccessCtx, "provider", nil)
	s.NoError(err, "expected no error trying to read with Access permissions")
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

	actualGroups, err := s.dataStore.Walk(s.hasWriteAccessCtx, "authProvider1", attributes)
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
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	grp := &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Add(s.hasWriteCtx, grp)
	s.NoError(err, "expected no error trying to write with permissions")

	grp = &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err = s.dataStore.Add(s.hasWriteAccessCtx, grp)
	s.NoError(err, "expected no error trying to write with Access permissions")
}

func (s *groupDataStoreTestSuite) TestEnforcesUpdate() {
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)

	grp := &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Update(s.hasNoneCtx, grp)
	s.Error(err, "expected an error trying to write without permissions")

	grp = &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err = s.dataStore.Update(s.hasReadCtx, grp)
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsUpdate() {
	s.expectGet(2, fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW))
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	grp := &storage.Group{Props: &storage.GroupProperties{
		Id:             "1",
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Update(s.hasWriteCtx, grp)
	s.NoError(err, "expected no error trying to write with permissions")

	grp = &storage.Group{Props: &storage.GroupProperties{
		Id:             "1",
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err = s.dataStore.Update(s.hasWriteAccessCtx, grp)
	s.NoError(err, "expected no error trying to write with Access permissions")
}

// TODO(ROX-11592): This can be removed once retrieving the group by its properties is fully deprecated.
func (s *groupDataStoreTestSuite) TestCanUpdateGroupByProps() {
	group := fixtures.GetGroups()[6]

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc([]*storage.Group{group})) // Make sure it can find the group

	updatedGroup := group.Clone()
	updatedGroup.GetProps().Id = "" // no id
	updatedGroup.RoleName = updatedGroup.GetRoleName() + "-updated"

	expectedGroup := updatedGroup.Clone()
	expectedGroup.GetProps().Id = group.GetProps().GetId()

	// It should upsert with the updated value
	s.storage.EXPECT().Upsert(gomock.Any(), expectedGroup).Return(nil)
	s.expectGet(1, fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW))

	s.NoError(s.dataStore.Update(s.hasWriteAccessCtx, updatedGroup.Clone()))
}

func (s *groupDataStoreTestSuite) TestEnforcesMutate() {
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Times(0)
	s.storage.EXPECT().DeleteMany(gomock.Any(), gomock.Any()).Times(0)

	grp := &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Mutate(s.hasNoneCtx, false, []*storage.Group{groupWithID}, []*storage.Group{groupWithID},
		[]*storage.Group{grp})
	s.Error(err, "expected an error trying to write without permissions")

	grp = &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err = s.dataStore.Mutate(s.hasReadCtx, false, []*storage.Group{groupWithID}, []*storage.Group{groupWithID},
		[]*storage.Group{grp})
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsMutate() {
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil).Times(4) // two calls * two operations (add, update)
	s.storage.EXPECT().DeleteMany(gomock.Any(), gomock.Any()).Return(nil).Times(2)
	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW), true, nil).Times(4)

	grp := &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err := s.dataStore.Mutate(s.hasWriteCtx, false, []*storage.Group{groupWithID}, []*storage.Group{groupWithID},
		[]*storage.Group{grp})
	s.NoError(err, "expected no error trying to write with permissions")

	grp = &storage.Group{Props: &storage.GroupProperties{
		AuthProviderId: "123",
		Key:            "123",
		Value:          "123",
	}, RoleName: "123"}
	err = s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{groupWithID}, []*storage.Group{groupWithID},
		[]*storage.Group{grp})
	s.NoError(err, "expected no error trying to write with Access permissions")
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
	s.expectGet(2, fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW))
	gomock.InOrder(
		s.storage.EXPECT().UpsertMany(gomock.Any(), toAdd),
		s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{toUpdate}),
		s.storage.EXPECT().DeleteMany(gomock.Any(), []string{toRemove.GetProps().GetId()}),
	)

	s.NoError(s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{toRemove}, []*storage.Group{toUpdate}, toAdd))
}

// TODO(ROX-11592): This can be removed once retrieving the group by its properties is fully deprecated.
func (s *groupDataStoreTestSuite) TestCanMutateGroupByProps() {
	groups := fixtures.GetGroups()

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc(groups)).Times(2)

	updatedGroup := groups[6].Clone()
	updatedGroup.GetProps().Id = "" // no id
	updatedGroup.RoleName = updatedGroup.GetRoleName() + "-updated"

	expectedGroup := updatedGroup.Clone()
	expectedGroup.GetProps().Id = groups[6].GetProps().GetId()

	removedGroup := groups[3].Clone()
	removedGroup.GetProps().Id = ""

	// It should upsert with the updated value
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{expectedGroup}).Return(nil)
	s.storage.EXPECT().DeleteMany(gomock.Any(), []string{groups[3].GetProps().GetId()}).Return(nil)
	s.expectGet(2, fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW))

	s.NoError(s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{removedGroup.Clone()}, []*storage.Group{updatedGroup.Clone()}, []*storage.Group{}))
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
				err := s.dataStore.Add(s.hasWriteAccessCtx, c.groupToAdd.Clone())
				s.Error(err)
				s.ErrorIs(err, errox.AlreadyExists)

				// Validate Mutate with additions returns an error if duplicate default group
				s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil).Times(0)
				err = s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{}, []*storage.Group{}, []*storage.Group{c.groupToAdd.Clone()})
				s.Error(err)
				s.ErrorIs(err, errox.AlreadyExists)
			} else {
				// Validate Add doesn't error if it's a new default
				s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				s.NoError(s.dataStore.Add(s.hasWriteAccessCtx, c.groupToAdd.Clone()))

				// Validate  Mutate with additions doesn't error if it's a new default
				s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				s.NoError(s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{}, []*storage.Group{}, []*storage.Group{c.groupToAdd.Clone()}))
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
	s.expectGet(2, fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW))
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc([]*storage.Group{initialGroup, defaultGroup})).Times(2)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Times(0)     // No update should happen
	s.storage.EXPECT().UpsertMany(gomock.Any(), gomock.Any()).Times(0) // No updates should happen

	// Unset Key / Value fields, making it a default group.
	updatedGroup := initialGroup.Clone()
	updatedGroup.GetProps().Key = ""
	updatedGroup.GetProps().Value = ""

	// Ensure a "AlreadyExists" error is yielded when trying to update the group.
	err := s.dataStore.Update(s.hasWriteAccessCtx, updatedGroup.Clone())
	s.Error(err)
	s.ErrorIs(err, errox.AlreadyExists)

	// Ensure a "AlreadyExists" error is yielded when trying to update the group using Mutate.
	err = s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{}, []*storage.Group{updatedGroup.Clone()}, []*storage.Group{})
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

	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(walkMockFunc([]*storage.Group{initialGroup, defaultGroup})).AnyTimes()

	// 1. Updating the default group's role should work.
	defaultGroup.RoleName = "non-admin" // Using the same defaultGroup object so that the Walk closure is also updated correctly

	s.storage.EXPECT().Upsert(gomock.Any(), defaultGroup)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{defaultGroup})

	s.expectGet(2, fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW))
	s.NoError(s.dataStore.Update(s.hasWriteAccessCtx, defaultGroup))
	s.NoError(s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{}, []*storage.Group{defaultGroup}, []*storage.Group{}))

	// 2. Update the default group to a non-default group.
	defaultGroup.GetProps().Key = "email" // Update the properties to make it a non-default group.
	defaultGroup.GetProps().Value = "test@example.com"

	s.storage.EXPECT().Upsert(gomock.Any(), defaultGroup)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{defaultGroup})

	s.expectGet(2, fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW))
	s.NoError(s.dataStore.Update(s.hasWriteAccessCtx, defaultGroup))
	s.NoError(s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{}, []*storage.Group{defaultGroup}, []*storage.Group{}))

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

	s.NoError(s.dataStore.Add(s.hasWriteAccessCtx, newDefaultGroup.Clone()))
	s.NoError(s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{}, []*storage.Group{}, []*storage.Group{newDefaultGroup}))
}

func (s *groupDataStoreTestSuite) TestEnforcesRemove() {
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)

	err := s.dataStore.Remove(s.hasNoneCtx, false, groupWithID.GetProps())
	s.Error(err, "expected an error trying to write without permissions")

	err = s.dataStore.Remove(s.hasReadCtx, false, groupWithID.GetProps())
	s.Error(err, "expected an error trying to write without permissions")
}

func (s *groupDataStoreTestSuite) TestAllowsRemove() {
	s.expectGet(2, fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW))
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	err := s.dataStore.Remove(s.hasWriteCtx, false, groupWithID.GetProps())
	s.NoError(err, "expected no error trying to write with permissions")

	err = s.dataStore.Remove(s.hasWriteAccessCtx, false, groupWithID.GetProps())
	s.NoError(err, "expected no error trying to write with Access permissions")
}

func (s *groupDataStoreTestSuite) expectGet(times int, group *storage.Group) *gomock.Call {
	return s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(group, true, nil).Times(times)
}

// TODO(ROX-11592): This can be removed once retrieving the group by its properties is fully deprecated.
func (s *groupDataStoreTestSuite) TestRemoveDeletesByProps() {
	groups := fixtures.GetGroups()
	expectedProps := groups[6].GetProps().Clone()
	expectedProps.Id = "" // clear out the id for the purposes of this test
	s.storage.EXPECT().Walk(gomock.Any(), gomock.Any()).Times(2).DoAndReturn(walkMockFunc(groups))

	// Test that removing when no groups match expected props fails
	s.Error(s.dataStore.Remove(s.hasWriteAccessCtx, false, &storage.GroupProperties{AuthProviderId: "i-don't-exist"}))

	// Test that can remove by props only
	s.expectGet(1, fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW))
	s.storage.EXPECT().Delete(gomock.Any(), groups[6].GetProps().GetId())
	err := s.dataStore.Remove(s.hasWriteAccessCtx, false, expectedProps)
	s.NoError(err)
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
		err := s.dataStore.Add(s.hasWriteAccessCtx, g)
		s.Error(err)
		s.ErrorIs(err, errox.InvalidArgs)

		err = s.dataStore.Update(s.hasWriteAccessCtx, g)
		s.Error(err)
		s.ErrorIs(err, errox.InvalidArgs)

		err = s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{g}, nil, nil)
		s.Error(err)
		s.ErrorIs(err, errox.InvalidArgs)

		err = s.dataStore.Mutate(s.hasWriteAccessCtx, false, nil, []*storage.Group{g}, nil)
		s.Error(err)
		s.ErrorIs(err, errox.InvalidArgs)

		err = s.dataStore.Mutate(s.hasWriteAccessCtx, false, nil, nil, []*storage.Group{g})
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
				MutabilityMode: storage.MutabilityMode_ALLOW,
			},
		},
	}, true, nil).Times(1)
	s.storage.EXPECT().Upsert(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.Update(s.hasWriteCtx, &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "id",
			AuthProviderId: "apid",
			Traits: &storage.Traits{
				MutabilityMode: storage.MutabilityMode_ALLOW_FORCED,
			},
			Key:   "abc",
			Value: "dfg",
		},
		RoleName: "Admin",
	})
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestUpdateImmutableError() {
	expectedGroup := fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW_FORCED)

	updatedGroup := expectedGroup.Clone()
	updatedGroup.GetProps().Key = ""
	updatedGroup.GetProps().Value = ""

	err := s.dataStore.Update(s.hasWriteCtx, updatedGroup)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *groupDataStoreTestSuite) TestDeleteImmutableNoForce() {
	expectedGroup := fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW_FORCED)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedGroup, true, nil).Times(1)

	err := s.dataStore.Remove(s.hasWriteCtx, false, expectedGroup.GetProps())
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *groupDataStoreTestSuite) TestDeleteImmutableForce() {
	expectedGroup := fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW_FORCED)

	s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(expectedGroup, true, nil).Times(1)
	s.storage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	err := s.dataStore.Remove(s.hasWriteCtx, true, expectedGroup.GetProps())
	s.NoError(err)
}

func (s *groupDataStoreTestSuite) TestDefaultGroupCannotBeImmutable() {
	group := &storage.Group{
		Props: &storage.GroupProperties{
			Id:             "id",
			AuthProviderId: "apid",
			Traits: &storage.Traits{
				MutabilityMode: storage.MutabilityMode_ALLOW_FORCED,
			},
		},
	}
	err := s.dataStore.Update(s.hasWriteCtx, group)
	s.ErrorIs(err, errox.InvalidArgs)
	err = s.dataStore.Add(s.hasWriteCtx, group)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *groupDataStoreTestSuite) TestMutateGroupNoForce() {
	mutableGroup := fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW)
	immutableGroup := fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW_FORCED)

	// 1. Try and remove an immutable group via mutate without force. This should fail.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mutableGroup, true, nil),
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(immutableGroup, true, nil),
	)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{mutableGroup}).Return(nil)
	err := s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{immutableGroup}, []*storage.Group{mutableGroup}, nil)
	s.Error(err)
	s.ErrorIs(err, errox.InvalidArgs)

	// 2. Try and update an immutable group via mutate without force. This should fail.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(immutableGroup, true, nil),
	)

	err = s.dataStore.Mutate(s.hasWriteAccessCtx, false, []*storage.Group{mutableGroup}, []*storage.Group{immutableGroup}, nil)
	s.Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
}

func (s *groupDataStoreTestSuite) TestMutateGroupForce() {
	mutableGroup := fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW)
	immutableGroup := fixtures.GetGroupWithMutability(storage.MutabilityMode_ALLOW_FORCED)

	// 1. Try and remove an immutable group via mutate with force.
	gomock.InOrder(
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(mutableGroup, true, nil),
		s.storage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(immutableGroup, true, nil),
	)
	s.storage.EXPECT().UpsertMany(gomock.Any(), []*storage.Group{mutableGroup}).Return(nil).Times(1)
	s.storage.EXPECT().DeleteMany(gomock.Any(), []string{immutableGroup.GetProps().GetId()}).Return(nil).Times(1)
	err := s.dataStore.Mutate(s.hasWriteAccessCtx, true, []*storage.Group{immutableGroup}, []*storage.Group{mutableGroup}, nil)
	s.NoError(err)
}
