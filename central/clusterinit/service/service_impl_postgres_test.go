//go:build sql_integration

package service

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusterinit/backend"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stretchr/testify/suite"
)

func TestServiceWithDatabase(t *testing.T) {
	suite.Run(t, new(clusterInitServicePostgresTestSuite))
}

type clusterInitServicePostgresTestSuite struct {
	suite.Suite

	db *pgtest.TestPostgres

	service Service
}

func (s *clusterInitServicePostgresTestSuite) SetupSuite() {
	s.db = pgtest.ForT(s.T())
	serviceBackend, err := backend.GetTestPostgresBackend(s.T(), s.db)
	s.Require().NoError(err)
	clusterStore, err := datastore.GetTestPostgresDataStore(s.T(), s.db)
	s.Require().NoError(err)
	s.service = New(serviceBackend, clusterStore)
}

func (s *clusterInitServicePostgresTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *clusterInitServicePostgresTestSuite) TestGetCAConfigSAC() {
	testCases := []struct {
		name        string
		ctx         context.Context
		expectedErr error
	}{
		{
			name:        "Users with full access should be able to retrieve CA Config",
			ctx:         sac.WithAllAccess(context.Background()),
			expectedErr: nil,
		},
		{
			name: "Users with Administration AND Integration read permissions should be able to retrieve CA Config",
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Administration, resources.Integration),
				),
			),
			expectedErr: nil,
		},
		{
			name: "Users with Administration permission only should NOT be able to retrieve CA Config",
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Administration),
				),
			),
			expectedErr: errox.NotAuthorized,
		},
		{
			name: "Users with Integration permission only should NOT be able to retrieve CA Config",
			ctx: sac.WithGlobalAccessScopeChecker(context.Background(),
				sac.AllowFixedScopes(
					sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
					sac.ResourceScopeKeys(resources.Integration),
				),
			),
			expectedErr: errox.NotAuthorized,
		},
		{
			name:        "Users with no access should NOT be able to retrieve CA Config",
			ctx:         sac.WithNoAccess(context.Background()),
			expectedErr: errox.NotAuthorized,
		},
	}

	for _, c := range testCases {
		s.Run(c.name, func() {
			result, err := s.service.GetCAConfig(c.ctx, &v1.Empty{})
			s.ErrorIs(err, c.expectedErr)
			if c.expectedErr != nil {
				s.NotNil(result)
			} else {
				s.Nil(result)
			}
		})
	}
}
