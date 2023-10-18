//go:build sql_integration

package datastore

import (
	"context"
	"testing"

	pgStore "github.com/stackrox/rox/central/auth/store/postgres"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	permissionSetPostgresStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	rolePostgresStore "github.com/stackrox/rox/central/role/store/role/postgres"
	accessScopePostgresStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

const (
	testRole1 = "New-Admin"
	testRole2 = "Super-Admin"
	testRole3 = "Super Continuous Integration"
)

var (
	existingRoles = set.NewFrozenStringSet(testRole1, testRole2, testRole3)
)

func TestAuthDatastorePostgres(t *testing.T) {
	suite.Run(t, new(datastorePostgresTestSuite))
}

type datastorePostgresTestSuite struct {
	suite.Suite

	ctx           context.Context
	pool          *pgtest.TestPostgres
	authDataStore DataStore
	roleDataStore roleDataStore.DataStore
}

func (s *datastorePostgresTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access),
		),
	)

	s.pool = pgtest.ForT(s.T())
	s.Require().NotNil(s.pool)

	store := pgStore.New(s.pool.DB)
	s.authDataStore = New(store)

	permSetStore := permissionSetPostgresStore.New(s.pool.DB)
	accessScopeStore := accessScopePostgresStore.New(s.pool.DB)
	roleStore := rolePostgresStore.New(s.pool.DB)
	s.roleDataStore = roleDataStore.New(roleStore, permSetStore, accessScopeStore, func(_ context.Context, _ func(*storage.Group) bool) ([]*storage.Group, error) {
		return nil, nil
	})
	s.addRoles()
}

func (s *datastorePostgresTestSuite) TearDownTest() {
	s.pool.Teardown(s.T())
	s.pool.Close()
}

func (s *datastorePostgresTestSuite) TestDeleteFKConstraint() {
	config, err := s.authDataStore.AddAuthM2MConfig(s.ctx, &storage.AuthMachineToMachineConfig{
		Id:                      "80c053c2-24a7-4b97-bd69-85b3a511241e",
		Type:                    storage.AuthMachineToMachineConfig_GITHUB_ACTIONS,
		TokenExpirationDuration: "5m",
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:   "sub",
				Value: "some-value",
				Role:  testRole1,
			},
		},
	})
	s.Require().NoError(err)

	s.Error(s.roleDataStore.RemoveRole(s.ctx, testRole1))

	s.NoError(s.authDataStore.RemoveAuthM2MConfig(s.ctx, config.GetId()))

	s.NoError(s.roleDataStore.RemoveRole(s.ctx, testRole1))
}

func (s *datastorePostgresTestSuite) addRoles() {
	permSetID := uuid.NewV4().String()
	accessScopeID := uuid.NewV4().String()
	s.Require().NoError(s.roleDataStore.AddPermissionSet(s.ctx, &storage.PermissionSet{
		Id:          permSetID,
		Name:        "test permission set",
		Description: "test permission set",
		ResourceToAccess: map[string]storage.Access{
			resources.Access.String(): storage.Access_READ_ACCESS,
		},
	}))
	s.Require().NoError(s.roleDataStore.AddAccessScope(s.ctx, &storage.SimpleAccessScope{
		Id:          accessScopeID,
		Name:        "test access scope",
		Description: "test access scope",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster-a"},
		},
	}))

	for _, role := range existingRoles.AsSlice() {
		s.Require().NoError(s.roleDataStore.AddRole(s.ctx, &storage.Role{
			Name:            role,
			Description:     "test role",
			PermissionSetId: permSetID,
			AccessScopeId:   accessScopeID,
		}))
	}
}
