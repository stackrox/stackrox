//go:build sql_integration

package updater

import (
	"context"
	"testing"

	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/policycleaner"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/errox"
	mockIntegrationHealth "github.com/stackrox/rox/pkg/integrationhealth/mocks"
	mockNotifierProcessor "github.com/stackrox/rox/pkg/notifier/mocks"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestNotifierUpdater(t *testing.T) {
	suite.Run(t, new(notifierUpdaterTestSuite))
}

type notifierUpdaterTestSuite struct {
	suite.Suite

	ctx     context.Context
	pgTest  *pgtest.TestPostgres
	updater *notifierUpdater
	nds     notifierDS.DataStore
	mp      *mockNotifierProcessor.MockProcessor
	mi      *mockIntegrationHealth.MockReporter
}

func (s *notifierUpdaterTestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access, resources.Integration),
		),
	)
	s.ctx = declarativeconfig.WithModifyDeclarativeOrImperative(s.ctx)

	s.pgTest = pgtest.ForT(s.T())
	s.Require().NotNil(s.pgTest)

	s.nds = notifierDS.GetTestPostgresDataStore(s.T(), s.pgTest.DB)
	s.mp = mockNotifierProcessor.NewMockProcessor(gomock.NewController(s.T()))
	s.mi = mockIntegrationHealth.NewMockReporter(gomock.NewController(s.T()))

	notifiers.Add(notifiers.GenericType, func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		return nil, nil
	})

	s.updater = newNotifierUpdater(s.nds, policycleaner.GetTestPolicyCleaner(s.T(), &mockDetectionSet{}), s.mp,
		declarativeConfigHealth.GetTestPostgresDataStore(s.T(), s.pgTest.DB), s.mi).(*notifierUpdater)
}

func (s *notifierUpdaterTestSuite) TearDownTest() {
	s.pgTest.Teardown(s.T())
	s.pgTest.Close()
}

func (s *notifierUpdaterTestSuite) TestUpsert() {
	cases := map[string]struct {
		mockCalls func()
		m         protocompat.Message
		err       error
	}{
		"invalid message type should yield an error": {
			m:   &storage.Role{Name: "something"},
			err: errox.InvariantViolation,
		},
		"valid message type should be upserted": {
			mockCalls: func() {
				gomock.InOrder(
					s.mp.EXPECT().UpdateNotifier(s.ctx, gomock.Any()),
					s.mi.EXPECT().Register("61a68f2a-2599-5a9f-a98a-8fc83e2c06cf", "testing",
						storage.IntegrationHealth_NOTIFIER).Return(nil),
				)
			},
			m: &storage.Notifier{
				Id:         "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
				Name:       "testing",
				Type:       "generic",
				UiEndpoint: "localhost:8000",
				Config: &storage.Notifier_Generic{Generic: &storage.Generic{
					Endpoint: "localhost:8000",
				}},
				Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
			},
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			if tc.mockCalls != nil {
				tc.mockCalls()
			}
			err := s.updater.Upsert(s.ctx, tc.m)
			s.ErrorIs(err, tc.err)
			if tc.err == nil {
				_, exists, err := s.updater.notifierDS.GetNotifier(s.ctx, s.updater.idExtractor(tc.m))
				s.NoError(err)
				s.True(exists)
			}
		})
	}
}

func (s *notifierUpdaterTestSuite) TestDelete_Successful() {
	id, err := s.updater.notifierDS.AddNotifier(s.ctx, &storage.Notifier{
		Id:         "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
		Name:       "testing",
		Type:       "generic",
		UiEndpoint: "localhost:8000",
		Config: &storage.Notifier_Generic{Generic: &storage.Generic{
			Endpoint: "localhost:8000",
		}},
		Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	})
	s.Require().NoError(err)

	gomock.InOrder(
		s.mp.EXPECT().RemoveNotifier(s.ctx, id),
		s.mi.EXPECT().RemoveIntegrationHealth(id),
	)

	names, err := s.updater.DeleteResources(s.ctx)
	s.NoError(err)
	s.Empty(names)

	notifier, exists, err := s.updater.notifierDS.GetNotifier(s.ctx, id)
	s.NoError(err)
	s.False(exists)
	s.Nil(notifier)
}

func (s *notifierUpdaterTestSuite) TestDelete_Error() {
	cases := map[string]struct {
		mockCalls func(id string)
		err       error
	}{
		"fail to delete notifier from policies": {
			mockCalls: func(id string) {
				s.updater.policyCleaner = policycleaner.GetTestPolicyCleaner(s.T(), &mockDetectionSet{fail: true})
			},
			err: errox.InvalidArgs,
		},
		"fail to remove integration health": {
			mockCalls: func(id string) {
				s.mp.EXPECT().RemoveNotifier(s.ctx, id)
				s.mi.EXPECT().RemoveIntegrationHealth(id).Return(errox.InvalidArgs)
			},
			err: errox.InvalidArgs,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			id, err := s.updater.notifierDS.AddNotifier(s.ctx, &storage.Notifier{
				Id:         "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
				Name:       "testing",
				Type:       "generic",
				UiEndpoint: "localhost:8000",
				Config: &storage.Notifier_Generic{Generic: &storage.Generic{
					Endpoint: "localhost:8000",
				}},
				Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
			})
			s.Require().NoError(err)
			s.Require().NoError(s.updater.healthDS.UpsertDeclarativeConfig(s.ctx, &storage.DeclarativeConfigHealth{
				Id:     id,
				Name:   "testing",
				Status: storage.DeclarativeConfigHealth_HEALTHY,
			}))

			tc.mockCalls(id)

			names, err := s.updater.DeleteResources(s.ctx)
			s.Contains(names, id)
			s.ErrorIs(err, tc.err)

			health, exists, err := s.updater.healthDS.GetDeclarativeConfig(s.ctx, id)
			s.NoError(err)
			s.True(exists)
			s.Equal(storage.DeclarativeConfigHealth_UNHEALTHY, health.GetStatus())
		})
	}
}

type mockDetectionSet struct {
	detection.PolicySet
	fail bool
}

func (m *mockDetectionSet) RemoveNotifier(_ string) error {
	if m.fail {
		return errox.InvalidArgs
	}
	return nil
}
