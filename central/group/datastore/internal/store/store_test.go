package store

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

func TestGroupStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(GroupStoreTestSuite))
}

type GroupStoreTestSuite struct {
	suite.Suite

	db *bolt.DB

	sto Store
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

func (s *GroupStoreTestSuite) TestAdd() {
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
		s.NoError(s.sto.Add(a))
	}

	for _, a := range groups {
		s.Error(s.sto.Add(a))
	}

	for _, a := range groups {
		full, err := s.sto.Get(a.GetProps())
		s.NoError(err)
		s.Equal(a, full)
	}

	retrievedGroups, err := s.sto.GetAll()
	s.NoError(err)
	s.ElementsMatch(groups, retrievedGroups)

	for _, a := range groups {
		s.NoError(s.sto.Remove(a.GetProps()))
	}

	groupsAfterDelete, err := s.sto.GetAll()
	s.NoError(err)
	s.Empty(groupsAfterDelete)
}

func (s *GroupStoreTestSuite) TestUpdate() {
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
		s.Error(s.sto.Update(a))
	}

	for _, a := range groups {
		s.NoError(s.sto.Add(a))
	}

	for _, a := range groups {
		s.NoError(s.sto.Update(a))
	}

	for _, a := range groups {
		full, err := s.sto.Get(a.GetProps())
		s.NoError(err)
		s.Equal(a, full)
	}

	retrievedGroups, err := s.sto.GetAll()
	s.NoError(err)
	s.ElementsMatch(groups, retrievedGroups)

	for _, a := range groups {
		s.NoError(s.sto.Remove(a.GetProps()))
	}

	groupsAfterDelete, err := s.sto.GetAll()
	s.NoError(err)
	s.Empty(groupsAfterDelete)
}

func (s *GroupStoreTestSuite) TestMutate() {
	startingState := []*storage.Group{
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

	for _, a := range startingState {
		s.NoError(s.sto.Add(a))
	}

	toRemove := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsCaptain",
				Id:             "1",
			},
			RoleName: "captain",
		},
	}

	toUpdate := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsAlsoCaptain",
				Id:             "2",
			},
			RoleName: "notcaptain",
		},
	}

	toAdd := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "DifferentAttribute",
				Value:          "IsNotCaptain",
				Id:             "4",
			},
			RoleName: "notcaptain",
		},
	}

	s.NoError(s.sto.Mutate(toRemove, toUpdate, toAdd))

	// Last starting state should be untouched.
	remainingStart, err := s.sto.Get(startingState[2].GetProps())
	s.NoError(err)
	s.Equal(startingState[2], remainingStart)

	// Removed starting state should not be present.
	for _, a := range toRemove {
		full, err := s.sto.Get(a.GetProps())
		s.NoError(err)
		s.Equal((*storage.Group)(nil), full)
	}

	// Updated value check.
	for _, a := range toUpdate {
		full, err := s.sto.Get(a.GetProps())
		s.NoError(err)
		s.Equal(a, full)
	}

	// Added value check.
	for _, a := range toAdd {
		full, err := s.sto.Get(a.GetProps())
		s.NoError(err)
		s.Equal(a, full)
	}

	// Remove all remaining groups, should be 3 (starting state had one added and one removed).
	retrievedGroups, err := s.sto.GetAll()
	s.NoError(err)
	s.Equal(3, len(retrievedGroups))
	for _, a := range retrievedGroups {
		s.NoError(s.sto.Remove(a.GetProps()))
	}
}

func (s *GroupStoreTestSuite) TestWalk() {
	groups := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				Id: "0",
			},
			RoleName: "role1",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Id:             "1",
			},
			RoleName: "role2",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Id:             "2",
			},
			RoleName: "role3",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Value:          "Value1",
				Id:             "3",
			},
			RoleName: "role4",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute2",
				Value:          "Value1",
				Id:             "4",
			},
			RoleName: "role5",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvide2",
				Key:            "Attribute1",
				Value:          "Value1",
				Id:             "5",
			},
			RoleName: "role6",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvide2",
				Key:            "Attribute2",
				Value:          "Value1",
				Id:             "6",
			},
			RoleName: "role7",
		},
	}

	expectedGroups := []*storage.Group{
		groups[1],
		groups[2],
		groups[3],
	}

	for _, a := range groups {
		s.NoError(s.sto.Add(a))
	}

	actualGroups, err := s.sto.Walk("authProvider1", map[string][]string{
		"Attribute1": {
			"Value1",
		},
		"Attribute2": {
			"Value2",
		},
	})
	s.NoError(err)
	s.ElementsMatch(expectedGroups, actualGroups)

	for _, a := range groups {
		s.NoError(s.sto.Remove(a.GetProps()))
	}
}

func (s *GroupStoreTestSuite) TestGetAll() {
	groups := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				Id: "0",
			},
			RoleName: "role1",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Id:             "1",
			},
			RoleName: "role2",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Id:             "2",
			},
			RoleName: "role3",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Value:          "Value1",
				Id:             "3",
			},
			RoleName: "role4",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute2",
				Value:          "Value1",
				Id:             "4",
			},
			RoleName: "role5",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvide2",
				Key:            "Attribute1",
				Value:          "Value1",
				Id:             "5",
			},
			RoleName: "role6",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvide2",
				Key:            "Attribute2",
				Value:          "Value1",
				Id:             "7",
			},
			RoleName: "role7",
		},
	}

	for _, a := range groups {
		s.NoError(s.sto.Add(a))
	}

	actualGroups, err := s.sto.GetAll()
	s.NoError(err)
	s.ElementsMatch(groups, actualGroups)

	actualGroups, err = s.sto.GetFiltered(nil)
	s.NoError(err)
	s.ElementsMatch(groups, actualGroups)

	actualGroups, err = s.sto.GetFiltered(func(*storage.GroupProperties) bool { return true })
	s.NoError(err)
	s.ElementsMatch(groups, actualGroups)
}

func (s *GroupStoreTestSuite) TestGetFiltered() {
	groups := []*storage.Group{
		{
			Props: &storage.GroupProperties{
				Id: "0",
			},
			RoleName: "role1",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Id:             "1",
			},
			RoleName: "role2",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Id:             "2",
			},
			RoleName: "role3",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Value:          "Value1",
				Id:             "3",
			},
			RoleName: "role4",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute2",
				Value:          "Value1",
				Id:             "4",
			},
			RoleName: "role5",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvide2",
				Key:            "Attribute1",
				Value:          "Value1",
				Id:             "5",
			},
			RoleName: "role6",
		},
		{
			Props: &storage.GroupProperties{
				AuthProviderId: "authProvide2",
				Key:            "Attribute2",
				Value:          "Value1",
				Id:             "6",
			},
			RoleName: "role7",
		},
	}

	for _, a := range groups {
		s.NoError(s.sto.Add(a))
	}

	actualGroups, err := s.sto.GetFiltered(func(*storage.GroupProperties) bool { return false })
	s.NoError(err)
	s.Empty(actualGroups)

	// Test with a selective filter
	actualGroups, err = s.sto.GetFiltered(func(props *storage.GroupProperties) bool {
		return props.GetAuthProviderId() == "authProvider1" || props.GetKey() == "Attribute2"
	})
	expectedGroups := []*storage.Group{
		groups[1], groups[2], groups[3], groups[4], groups[6],
	}
	s.NoError(err)
	s.ElementsMatch(expectedGroups, actualGroups)
}
