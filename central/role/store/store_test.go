package store

import (
	"os"
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestRoleStore(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(RoleStoreTestSuite))
}

type RoleStoreTestSuite struct {
	suite.Suite

	db *bbolt.DB

	sto Store
}

func (s *RoleStoreTestSuite) SetupSuite() {
	db, err := bolthelper.NewTemp(s.T().Name() + ".db")
	s.Require().NoError(err, "Failed to make BoltDB: %s", err)

	s.db = db
	s.sto = New(db)
}

func (s *RoleStoreTestSuite) TearDownSuite() {
	if s.db != nil {
		s.db.Close()
		os.Remove(s.db.Path())
	}
}

func (s *RoleStoreTestSuite) TestAdd() {
	roles := []*v1.Role{
		{
			Name: "ship",
			ResourceToAccess: map[string]v1.Access{
				"Policy": v1.Access_READ_ACCESS,
			},
		},
		{
			Name: "captain",
		},
		{
			Name: "squibbly",
		},
	}

	for _, a := range roles {
		s.NoError(s.sto.AddRole(a))
	}

	for _, a := range roles {
		s.Error(s.sto.AddRole(a))
	}

	for _, a := range roles {
		full, err := s.sto.GetRole(a.GetName())
		s.NoError(err)
		s.Equal(a, full)
	}

	retrievedRoles, err := s.sto.GetAllRoles()
	s.NoError(err)
	s.ElementsMatch(roles, retrievedRoles)

	for _, a := range roles {
		s.NoError(s.sto.RemoveRole(a.GetName()))
	}
}

func (s *RoleStoreTestSuite) TestUpdate() {
	roles := []*v1.Role{
		{
			Name: "ship",
			ResourceToAccess: map[string]v1.Access{
				"Policy": v1.Access_READ_ACCESS,
			},
		},
		{
			Name: "captain",
		},
		{
			Name: "squibbly",
		},
	}

	for _, a := range roles {
		s.Error(s.sto.UpdateRole(a))
	}

	for _, a := range roles {
		s.NoError(s.sto.AddRole(a))
	}

	for _, a := range roles {
		s.NoError(s.sto.UpdateRole(a))
	}

	for _, a := range roles {
		full, err := s.sto.GetRole(a.GetName())
		s.NoError(err)
		s.Equal(a, full)
	}

	retrievedRoles, err := s.sto.GetAllRoles()
	s.NoError(err)
	s.ElementsMatch(roles, retrievedRoles)

	for _, a := range roles {
		s.NoError(s.sto.RemoveRole(a.GetName()))
	}
}
