package store

import (
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/testutils"
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
		testutils.TearDownDB(s.db)
	}
}

func (s *RoleStoreTestSuite) TestAdd() {
	roles := []*storage.Role{
		{
			Name: "ship",
			ResourceToAccess: map[string]storage.Access{
				"Policy": storage.Access_READ_ACCESS,
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
	roles := []*storage.Role{
		{
			Name: "ship",
			ResourceToAccess: map[string]storage.Access{
				"Policy": storage.Access_READ_ACCESS,
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
