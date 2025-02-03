//go:build sql_integration

package updater

import (
	"context"
	"testing"

	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	roleDS "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestPermissionSetUpdater(t *testing.T) {
	suite.Run(t, new(permissionSetUpdaterTestSuite))
}

type permissionSetUpdaterTestSuite struct {
	suite.Suite

	ctx     context.Context
	pgTest  *pgtest.TestPostgres
	updater *permissionSetUpdater
}

func (s *permissionSetUpdaterTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access, resources.Integration),
		),
	)
	s.ctx = declarativeconfig.WithModifyDeclarativeOrImperative(s.ctx)

	s.pgTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.pgTest)
	rds, err := roleDS.GetTestPostgresDataStore(s.T(), s.pgTest.DB)
	s.Require().NoError(err)
	s.updater = newPermissionSetUpdater(rds, declarativeConfigHealth.GetTestPostgresDataStore(s.T(),
		s.pgTest.DB)).(*permissionSetUpdater)
}

func (s *permissionSetUpdaterTestSuite) TearDownTest() {
	s.pgTest.Teardown(s.T())
	s.pgTest.Close()
}

func (s *permissionSetUpdaterTestSuite) TestUpsert() {
	cases := map[string]struct {
		m   protocompat.Message
		err error
	}{
		"invalid message type should yield an error": {
			m:   &storage.Role{Name: "test"},
			err: errox.InvariantViolation,
		},
		"valid message type should be upserted": {
			m: &storage.PermissionSet{
				Id:          "04a87e34-b568-5e14-90ac-380d25c8689b",
				Name:        "testing",
				Description: "testing",
			},
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			err := s.updater.Upsert(s.ctx, tc.m)
			s.ErrorIs(err, tc.err)
			if tc.err == nil {
				_, exists, err := s.updater.roleDS.GetPermissionSet(s.ctx, s.updater.idExtractor(tc.m))
				s.NoError(err)
				s.True(exists)
			}
		})
	}
}

func (s *permissionSetUpdaterTestSuite) TestDelete_Successful() {
	s.Require().NoError(s.updater.roleDS.AddPermissionSet(s.ctx, &storage.PermissionSet{
		Id:          "04a87e34-b568-5e14-90ac-380d25c8689b",
		Name:        "testing",
		Description: "testing",
		Traits:      &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))

	names, deletionCount, err := s.updater.DeleteResources(s.ctx)
	s.NoError(err)
	s.Empty(names)
	s.Equal(1, deletionCount)
}

func (s *permissionSetUpdaterTestSuite) TestDelete_Error() {
	s.Require().NoError(s.updater.roleDS.AddAccessScope(s.ctx, &storage.SimpleAccessScope{
		Id:          "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
		Name:        "testing",
		Description: "testing",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster1"},
		},
		Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))
	s.Require().NoError(s.updater.roleDS.AddPermissionSet(s.ctx, &storage.PermissionSet{
		Id:          "04a87e34-b568-5e14-90ac-380d25c8689b",
		Name:        "testing",
		Description: "testing",
		Traits:      &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))
	s.Require().NoError(s.updater.roleDS.AddRole(s.ctx, &storage.Role{
		Name:            "test",
		Description:     "test",
		PermissionSetId: "04a87e34-b568-5e14-90ac-380d25c8689b",
		AccessScopeId:   "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
		Traits:          &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))
	s.Require().NoError(s.updater.healthDS.UpsertDeclarativeConfig(s.ctx, &storage.DeclarativeConfigHealth{
		Id:     "04a87e34-b568-5e14-90ac-380d25c8689b",
		Name:   "testing",
		Status: storage.DeclarativeConfigHealth_HEALTHY,
	}))

	names, deletionCount, err := s.updater.DeleteResources(s.ctx)
	s.Contains(names, "04a87e34-b568-5e14-90ac-380d25c8689b")
	s.ErrorIs(err, errox.ReferencedByAnotherObject)
	s.Equal(0, deletionCount)

	health, exists, err := s.updater.healthDS.GetDeclarativeConfig(s.ctx,
		"04a87e34-b568-5e14-90ac-380d25c8689b")
	s.NoError(err)
	s.True(exists)
	s.Equal(storage.DeclarativeConfigHealth_UNHEALTHY, health.GetStatus())

	permSet, exists, err := s.updater.roleDS.GetPermissionSet(s.ctx, "04a87e34-b568-5e14-90ac-380d25c8689b")
	s.NoError(err)
	s.True(exists)
	s.Equal(permSet.GetTraits().GetOrigin(), storage.Traits_DECLARATIVE_ORPHANED)
}
