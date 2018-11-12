package store

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
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

func (s *GroupStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(s.T().Name() + ".db")
	s.Require().NoError(err, "Failed to make BoltDB: %s", err)

	s.db = db
	s.sto = New(db)
}

func (s *GroupStoreTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
		os.Remove(s.db.Path())
	}
}

func (s *GroupStoreTestSuite) TestAdd() {
	groups := []*v1.Group{
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsCaptain",
			},
			RoleName: "captain",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsAlsoCaptain",
			},
			RoleName: "captain",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "DifferentAttribute",
				Value:          "IsCaptain",
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
}

func (s *GroupStoreTestSuite) TestUpdate() {
	groups := []*v1.Group{
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsCaptain",
			},
			RoleName: "captain",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsAlsoCaptain",
			},
			RoleName: "captain",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "DifferentAttribute",
				Value:          "IsCaptain",
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
}

func (s *GroupStoreTestSuite) TestUpsert() {
	groups := []*v1.Group{
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsCaptain",
			},
			RoleName: "captain",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute",
				Value:          "IsAlsoCaptain",
			},
			RoleName: "captain",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "DifferentAttribute",
				Value:          "IsCaptain",
			},
			RoleName: "captain",
		},
	}

	for _, a := range groups {
		s.NoError(s.sto.Upsert(a))
	}

	for _, a := range groups {
		s.NoError(s.sto.Upsert(a))
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
}

func (s *GroupStoreTestSuite) TestWalk() {
	groups := []*v1.Group{
		{
			RoleName: "role1",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
			},
			RoleName: "role2",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
			},
			RoleName: "role3",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute1",
				Value:          "Value1",
			},
			RoleName: "role4",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvider1",
				Key:            "Attribute2",
				Value:          "Value1",
			},
			RoleName: "role5",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvide2",
				Key:            "Attribute1",
				Value:          "Value1",
			},
			RoleName: "role6",
		},
		{
			Props: &v1.GroupProperties{
				AuthProviderId: "authProvide2",
				Key:            "Attribute2",
				Value:          "Value1",
			},
			RoleName: "role7",
		},
	}

	expectedGroups := []*v1.Group{
		groups[0],
		groups[1],
		groups[2],
		groups[3],
	}

	for _, a := range groups {
		s.NoError(s.sto.Upsert(a))
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
