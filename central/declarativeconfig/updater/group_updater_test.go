//go:build sql_integration

package updater

import (
	"context"
	"testing"

	authProviderDS "github.com/stackrox/rox/central/authprovider/datastore"
	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	groupDS "github.com/stackrox/rox/central/group/datastore"
	roleDS "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestGroupUpdater(t *testing.T) {
	suite.Run(t, new(groupUpdaterTestSuite))
}

type groupUpdaterTestSuite struct {
	suite.Suite

	ctx     context.Context
	pgTest  *pgtest.TestPostgres
	updater *groupUpdater
	ads     authproviders.Store
	rds     roleDS.DataStore
}

func (s *groupUpdaterTestSuite) SetupTest() {
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
	s.rds = rds
	s.ads = authProviderDS.GetTestPostgresDataStore(s.T(), s.pgTest.DB)
	s.updater = newGroupUpdater(
		groupDS.GetTestPostgresDataStore(s.T(), s.pgTest.DB, s.rds, s.ads),
		declarativeConfigHealth.GetTestPostgresDataStore(s.T(), s.pgTest.DB)).(*groupUpdater)
}

func (s *groupUpdaterTestSuite) TearDownTest() {
	s.pgTest.Teardown(s.T())
	s.pgTest.Close()
}

func (s *groupUpdaterTestSuite) TestUpsert() {
	cases := map[string]struct {
		prepData func()
		m        protocompat.Message
		err      error
	}{
		"invalid message type should yield an error": {
			m:   &storage.PermissionSet{Id: "some-id"},
			err: errox.InvariantViolation,
		},
		"valid message type should be upserted": {
			prepData: func() {
				s.Require().NoError(s.rds.AddAccessScope(s.ctx, &storage.SimpleAccessScope{
					Id:          "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
					Name:        "testing",
					Description: "testing",
					Rules: &storage.SimpleAccessScope_Rules{
						IncludedClusters: []string{"cluster1"},
					},
					Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
				}))
				s.Require().NoError(s.rds.AddPermissionSet(s.ctx, &storage.PermissionSet{
					Id:          "04a87e34-b568-5e14-90ac-380d25c8689b",
					Name:        "testing",
					Description: "testing",
					Traits:      &storage.Traits{Origin: storage.Traits_DECLARATIVE},
				}))
				s.Require().NoError(s.rds.AddRole(s.ctx, &storage.Role{
					Name:            "test",
					Description:     "test",
					PermissionSetId: "04a87e34-b568-5e14-90ac-380d25c8689b",
					AccessScopeId:   "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
					Traits:          &storage.Traits{Origin: storage.Traits_DECLARATIVE},
				}))
				s.Require().NoError(s.ads.AddAuthProvider(s.ctx, &storage.AuthProvider{
					Id:         "4df1b98c-24ed-4073-a9ad-356aec6bb62d",
					Name:       "basic",
					Type:       "basic",
					UiEndpoint: "http://localhost",
					LoginUrl:   "sso/something",
					Traits:     &storage.Traits{Origin: storage.Traits_DECLARATIVE},
				}))
			},
			m: &storage.Group{
				Props: &storage.GroupProperties{
					AuthProviderId: "4df1b98c-24ed-4073-a9ad-356aec6bb62d",
					Key:            "",
					Value:          "",
				},
				RoleName: "test",
			},
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			if tc.prepData != nil {
				tc.prepData()
			}
			err := s.updater.Upsert(s.ctx, tc.m)
			s.ErrorIs(err, tc.err)
			if tc.err == nil {
				_, err := s.updater.groupDS.Get(s.ctx, &storage.GroupProperties{
					Id:             s.updater.idExtractor(tc.m),
					AuthProviderId: "4df1b98c-24ed-4073-a9ad-356aec6bb62d",
				})
				s.NoError(err)
			}
		})
	}
}

func (s *groupUpdaterTestSuite) TestDelete_Successful() {
	s.Require().NoError(s.rds.AddAccessScope(s.ctx, &storage.SimpleAccessScope{
		Id:          "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
		Name:        "testing",
		Description: "testing",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{"cluster1"},
		},
		Traits: &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))
	s.Require().NoError(s.rds.AddPermissionSet(s.ctx, &storage.PermissionSet{
		Id:          "04a87e34-b568-5e14-90ac-380d25c8689b",
		Name:        "testing",
		Description: "testing",
		Traits:      &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))
	s.Require().NoError(s.rds.AddRole(s.ctx, &storage.Role{
		Name:            "test",
		Description:     "test",
		PermissionSetId: "04a87e34-b568-5e14-90ac-380d25c8689b",
		AccessScopeId:   "61a68f2a-2599-5a9f-a98a-8fc83e2c06cf",
		Traits:          &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))
	s.Require().NoError(s.ads.AddAuthProvider(s.ctx, &storage.AuthProvider{
		Id:         "4df1b98c-24ed-4073-a9ad-356aec6bb62d",
		Name:       "basic",
		Type:       "basic",
		UiEndpoint: "http://localhost",
		LoginUrl:   "sso/something",
		Traits:     &storage.Traits{Origin: storage.Traits_DECLARATIVE},
	}))
	s.Require().NoError(s.updater.groupDS.Add(s.ctx, &storage.Group{
		Props: &storage.GroupProperties{
			AuthProviderId: "4df1b98c-24ed-4073-a9ad-356aec6bb62d",
			Key:            "",
			Value:          "",
		},
		RoleName: "test",
	}))

	names, err := s.updater.DeleteResources(s.ctx)
	s.NoError(err)
	s.Empty(names)
}
