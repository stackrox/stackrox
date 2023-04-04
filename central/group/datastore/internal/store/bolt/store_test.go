package bolt

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/group/datastore/internal/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

func TestGroupStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GroupStoreTestSuite))
}

type GroupStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	sto store.Store
}

func (s *GroupStoreTestSuite) SetupTest() {
	db, err := bolthelper.NewTemp(s.T().Name() + ".db")
	s.Require().NoError(err, "Failed to make BoltDB: %s", err)

	s.db = db
	s.sto = New(db)
}

func (s *GroupStoreTestSuite) TearDownTest() {
	if s.db != nil {
		testutils.TearDownDB(s.db)
	}
}

func (s *GroupStoreTestSuite) TestUpsertAddsGroups() {
	groups := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsCaptain",
				Id:             "1",
			},
			RoleName: "captain",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsAlsoCaptain",
				Id:             "2",
			},
			RoleName: "captain",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "DifferentAttribute",
				Value:          "IsCaptain",
				Id:             "3",
			},
			RoleName: "captain",
		},
	}

	for _, a := range groups {
		s.NoError(s.sto.Upsert(ctx, a)) // adding new group via upsert should work
	}

	for _, a := range groups {
		full, exists, err := s.sto.Get(ctx, a.GetProps().GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(a, full)
	}

	retrievedGroups, err := s.sto.GetAll(ctx)
	s.NoError(err)
	s.ElementsMatch(groups, retrievedGroups)

	for _, a := range groups {
		s.NoError(s.sto.Delete(ctx, a.GetProps().GetId()))
	}

	groupsAfterDelete, err := s.sto.GetAll(ctx)
	s.NoError(err)
	s.Empty(groupsAfterDelete)
}

func (s *GroupStoreTestSuite) TestUpsertUpdatesExistingGroups() {
	groups := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsCaptain",
				Id:             "1",
			},
			RoleName: "captain",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsAlsoCaptain",
				Id:             "2",
			},
			RoleName: "captain",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "DifferentAttribute",
				Value:          "IsCaptain",
				Id:             "3",
			},
			RoleName: "captain",
		},
	}

	for _, a := range groups {
		s.NoError(s.sto.Upsert(ctx, a)) // add first
	}

	for _, a := range groups {
		a.GetProps().Value = a.GetProps().GetValue() + "-updated"
		s.NoError(s.sto.Upsert(ctx, a)) // then update using the same upsert
	}

	for _, a := range groups {
		full, exists, err := s.sto.Get(ctx, a.GetProps().GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(a, full)
	}

	retrievedGroups, err := s.sto.GetAll(ctx)
	s.NoError(err)
	s.ElementsMatch(groups, retrievedGroups)

	for _, a := range groups {
		s.NoError(s.sto.Delete(ctx, a.GetProps().GetId()))
	}

	groupsAfterDelete, err := s.sto.GetAll(ctx)
	s.NoError(err)
	s.Empty(groupsAfterDelete)
}

func (s *GroupStoreTestSuite) TestUpsertMany() {
	groups := fixtures.GetGroups()

	s.NoError(s.sto.UpsertMany(ctx, groups)) // adding all new groups should work

	// Validate all groups got added
	for _, a := range groups {
		full, exists, err := s.sto.Get(ctx, a.GetProps().GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(a, full)
	}

	for _, g := range groups {
		g.RoleName = g.GetRoleName() + "-updated"
	}

	s.NoError(s.sto.UpsertMany(ctx, groups)) // updating all the groups should work

	// Validate all groups got added
	for _, a := range groups {
		full, exists, err := s.sto.Get(ctx, a.GetProps().GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(a, full)
	}

	retrievedGroups, err := s.sto.GetAll(ctx)
	s.NoError(err)
	s.ElementsMatch(groups, retrievedGroups)

	for _, a := range groups {
		s.NoError(s.sto.Delete(ctx, a.GetProps().GetId()))
	}
}

func (s *GroupStoreTestSuite) TestWalk() {
	groups := fixtures.GetGroups()

	for _, a := range groups {
		s.NoError(s.sto.Upsert(ctx, a))
	}

	var foundGroups []*storage.Group
	err := s.sto.Walk(ctx, func(g *storage.Group) error {
		foundGroups = append(foundGroups, g)
		return nil
	})
	s.NoError(err)
	s.ElementsMatch(groups, foundGroups)

	for _, a := range groups {
		s.NoError(s.sto.Delete(ctx, a.GetProps().GetId()))
	}
}

func (s *GroupStoreTestSuite) TestGetAll() {
	groups := fixtures.GetGroups()

	for _, a := range groups {
		s.NoError(s.sto.Upsert(ctx, a))
	}

	actualGroups, err := s.sto.GetAll(ctx)
	s.NoError(err)
	s.ElementsMatch(groups, actualGroups)
}

func (s *GroupStoreTestSuite) TestDelete() {
	startingState := fixtures.GetGroups()

	for _, a := range startingState {
		s.NoError(s.sto.Upsert(ctx, a))
	}

	for _, g := range startingState[:3] {
		s.NoError(s.sto.Delete(ctx, g.GetProps().GetId()))
	}

	// Trying to delete a group that doesn't exist (by id) should fail
	s.Error(s.sto.Delete(ctx, "this-id-does-not-exist"), "Expected error when trying to delete non-existent group")
	s.validateDeletes(startingState, 3)
}

func (s *GroupStoreTestSuite) TestDeleteMany() {
	startingState := fixtures.GetGroups()
	var startingStateIds []string

	for _, a := range startingState {
		s.NoError(s.sto.Upsert(ctx, a))
		startingStateIds = append(startingStateIds, a.GetProps().GetId())
	}

	s.NoError(s.sto.DeleteMany(ctx, startingStateIds[:3]))
	s.validateDeletes(startingState, 3)
}

func (s *GroupStoreTestSuite) TestDeleteManyRollsBackIfAnyFails() {
	startingState := fixtures.GetGroups()
	var startingStateIds []string

	for _, a := range startingState {
		s.NoError(s.sto.Upsert(ctx, a))
		startingStateIds = append(startingStateIds, a.GetProps().GetId())
	}
	startingStateIds = append(startingStateIds, "this-doesn't-exist") // add in an id that should error out

	err := s.sto.DeleteMany(ctx, startingStateIds)
	s.Error(err)
	s.ErrorIs(err, errox.NotFound)

	// All the other groups whose ids did exist shouldn't get deleted (i.e. txn rolled back)
	retrievedGroups, err := s.sto.GetAll(ctx)
	s.NoError(err)
	s.Len(retrievedGroups, len(startingState))

	// Cleanup
	for _, a := range retrievedGroups {
		s.NoError(s.sto.Delete(ctx, a.GetProps().GetId()))
	}
}

func (s *GroupStoreTestSuite) validateDeletes(startingState []*storage.Group, numRemoved int) {
	// Removed starting state should not be present.
	for _, a := range startingState[:numRemoved] {
		full, exists, err := s.sto.Get(ctx, a.GetProps().GetId())
		s.NoError(err)
		s.False(exists)
		s.Equal((*storage.Group)(nil), full)
	}

	// Remaining elements in starting state should be untouched.
	for _, g := range startingState[numRemoved:] {
		remainingStart, exists, err := s.sto.Get(ctx, g.GetProps().GetId())
		s.NoError(err)
		s.True(exists)
		s.Equal(g, remainingStart)
	}

	// Remove all remaining groups, should be len(startingState) - numRemoved
	retrievedGroups, err := s.sto.GetAll(ctx)
	s.NoError(err)
	s.Equal(len(startingState)-numRemoved, len(retrievedGroups))
	for _, a := range retrievedGroups {
		s.NoError(s.sto.Delete(ctx, a.GetProps().GetId()))
	}
}
