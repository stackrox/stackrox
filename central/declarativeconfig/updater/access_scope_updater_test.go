//go:build sql_integration

package updater

import (
	"context"
	"testing"

	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	roleDS "github.com/stackrox/rox/central/role/datastore"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestAccessScopeUpdater(t *testing.T) {
	suite.Run(t, new(updaterTestSuite))
}

type updaterTestSuite struct {
	suite.Suite

	ctx     context.Context
	pgTest  *pgtest.TestPostgres
	updater *accessScopeUpdater
}

func (s *updaterTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access, resources.Integration),
		),
	)
	s.ctx = declarativeconfig.WithModifyDeclarativeOrImperative(s.ctx)

	s.pgTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.pgTest)
	ds, err := roleDS.GetTestPostgresDataStore(s.T(), s.pgTest.DB)
	s.Require().NoError(err)
	s.updater = newAccessScopeUpdater(ds,
		declarativeConfigHealth.GetTestPostgresDataStore(s.T(), s.pgTest.DB)).(*accessScopeUpdater)
}

func (s *updaterTestSuite) TearDownTest() {
	s.pgTest.Teardown(s.T())
	s.pgTest.Close()
}

func (s *updaterTestSuite) TestUpsert() {
	cases := map[string]struct {
		m   protocompat.Message
		err error
	}{
		"invalid message type should yield an error": {
			m:   &storage.Role{Name: "something"},
			err: errox.InvariantViolation,
		},
		"valid message type should be upserted": {
			m: &storage.SimpleAccessScope{
				Id:          "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
				Name:        "testing",
				Description: "testing",
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: []string{"cluster1"},
				},
			},
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			err := s.updater.Upsert(s.ctx, tc.m)
			s.ErrorIs(err, tc.err)
			if tc.err == nil {
				_, exists, err := s.updater.roleDS.GetAccessScope(s.ctx, s.updater.idExtractor(tc.m))
				s.NoError(err)
				s.True(exists)
			}
		})
	}
}

func (s *updaterTestSuite) TestDelete_Successful() {
	scopes := []*storage.SimpleAccessScope{
		{
			Id:          "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
			Name:        "test-1",
			Description: "",
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{"cluster1"},
			},
			Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
		},
		{
			Id:          "04a87e34-b568-5e14-90ac-380d25c8689b",
			Name:        "test-2",
			Description: "",
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{"cluster1"},
			},
			Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
		},
		{
			Id:          "0925514f-3a33-5931-b431-756406e1a008",
			Name:        "test-3",
			Description: "",
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{"cluster1"},
			},
			Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
		},
	}

	for _, scope := range scopes {
		s.Require().NoError(s.updater.roleDS.AddAccessScope(s.ctx, scope))
	}

	failedIDs, err := s.updater.DeleteResources(s.ctx, "0925514f-3a33-5931-b431-756406e1a008")
	s.NoError(err)
	s.Empty(failedIDs)

	_, exists, err := s.updater.roleDS.GetAccessScope(s.ctx, scopes[0].GetId())
	s.False(exists)
	s.NoError(err)

	_, exists, err = s.updater.roleDS.GetAccessScope(s.ctx, scopes[1].GetId())
	s.False(exists)
	s.NoError(err)

	_, exists, err = s.updater.roleDS.GetAccessScope(s.ctx, scopes[2].GetId())
	s.True(exists)
	s.NoError(err)
}

func (s *updaterTestSuite) TestDelete_Error() {
	invalidErr := errox.InvalidArgs.New("something is wrong")
	referenceErr := errox.ReferencedByAnotherObject.New("something is referenced")

	m := roleMocks.NewMockDataStore(gomock.NewController(s.T()))
	scopes := []*storage.SimpleAccessScope{
		{
			Id:          "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
			Name:        "test-1",
			Description: "",
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{"cluster1"},
			},
			Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
		},
		{
			Id:          "04a87e34-b568-5e14-90ac-380d25c8689b",
			Name:        "test-2",
			Description: "",
			Rules: &storage.SimpleAccessScope_Rules{
				IncludedClusters: []string{"cluster1"},
			},
			Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
		},
	}
	orphanedScope := scopes[1].Clone()
	orphanedScope.Traits.Origin = storage.Traits_DECLARATIVE_ORPHANED

	healths := []*storage.DeclarativeConfigHealth{
		{
			Id:           "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
			Name:         "test-1",
			Status:       storage.DeclarativeConfigHealth_HEALTHY,
			ResourceName: "test-1",
			ResourceType: storage.DeclarativeConfigHealth_ACCESS_SCOPE,
		},
		{
			Id:           "04a87e34-b568-5e14-90ac-380d25c8689b",
			Name:         "test-2",
			Status:       storage.DeclarativeConfigHealth_HEALTHY,
			ResourceName: "test-2",
			ResourceType: storage.DeclarativeConfigHealth_ACCESS_SCOPE,
		},
	}

	for _, health := range healths {
		s.Require().NoError(s.updater.healthDS.UpsertDeclarativeConfig(s.ctx, health))
	}

	gomock.InOrder(
		m.EXPECT().GetAccessScopesFiltered(gomock.Any(), gomock.Any()).Return(scopes, nil),
		m.EXPECT().RemoveAccessScope(gomock.Any(), scopes[0].GetId()).Return(invalidErr),
		m.EXPECT().RemoveAccessScope(gomock.Any(), scopes[1].GetId()).Return(referenceErr),
		m.EXPECT().UpsertAccessScope(gomock.Any(), orphanedScope).Return(nil),
	)

	s.updater.roleDS = m

	failedIDs, err := s.updater.DeleteResources(s.ctx)
	s.Error(err)
	s.ElementsMatch([]string{scopes[0].GetId(), scopes[1].GetId()}, failedIDs)

	health, exists, err := s.updater.healthDS.GetDeclarativeConfig(s.ctx, scopes[0].GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(storage.DeclarativeConfigHealth_UNHEALTHY, health.GetStatus())
	s.Contains(health.GetErrorMessage(), invalidErr.Error())

	health, exists, err = s.updater.healthDS.GetDeclarativeConfig(s.ctx, scopes[1].GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(storage.DeclarativeConfigHealth_UNHEALTHY, health.GetStatus())
	s.Contains(health.GetErrorMessage(), referenceErr.Error())
}
