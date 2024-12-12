//go:build sql_integration

package search

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	"github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

type AlertsSearchSuite struct {
	suite.Suite

	testPostgres *pgtest.TestPostgres
	store        store.Store
	searcher     Searcher
}

func TestAlertsIndex(t *testing.T) {
	suite.Run(t, new(AlertsSearchSuite))
}

func (s *AlertsSearchSuite) SetupTest() {
	s.testPostgres = pgtest.ForT(s.T())

	s.store = postgres.New(s.testPostgres.DB)
	s.searcher = New(s.store)
}

func (s *AlertsSearchSuite) TearDownTest() {
	s.testPostgres.Teardown(s.T())
}

func (s *AlertsSearchSuite) TestSearch() {
	alert := fixtures.GetAlert()
	alert.EntityType = storage.Alert_DEPLOYMENT
	alert.PlatformComponent = false

	foundAlert, exists, err := s.store.Get(ctx, alert.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundAlert)

	s.NoError(s.store.Upsert(ctx, alert))
	foundAlert, exists, err = s.store.Get(ctx, alert.GetId())
	s.NoError(err)
	s.True(exists)
	protoassert.Equal(s.T(), alert, foundAlert)

	// Common alert searches
	results, err := s.searcher.Search(ctx, search.NewQueryBuilder().AddExactMatches(search.DeploymentID, alert.GetDeployment().GetId()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, alert.GetDeployment().GetId()).
		AddExactMatches(search.PolicyID, alert.GetPolicy().GetId()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		ProtoQuery()
	results, err = s.searcher.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)

	q = search.NewQueryBuilder().
		AddBools(search.PlatformComponent, false).
		AddExactMatches(search.EntityType, storage.Alert_DEPLOYMENT.String()).
		ProtoQuery()
	results, err = s.searcher.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)

	q = search.NewQueryBuilder().
		AddBools(search.PlatformComponent, true).
		ProtoQuery()
	results, err = s.searcher.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 0)
}
