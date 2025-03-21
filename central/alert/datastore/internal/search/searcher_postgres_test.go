//go:build sql_integration

package search

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/alert/datastore/internal/store"
	"github.com/stackrox/rox/central/alert/datastore/internal/store/postgres"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
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
	results, err := s.searcher.Search(ctx, search.NewQueryBuilder().AddExactMatches(search.DeploymentID, alert.GetDeployment().GetId()).ProtoQuery(), true)
	s.NoError(err)
	s.Len(results, 1)

	q := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, alert.GetDeployment().GetId()).
		AddExactMatches(search.PolicyID, alert.GetPolicy().GetId()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		ProtoQuery()
	results, err = s.searcher.Search(ctx, q, true)
	s.NoError(err)
	s.Len(results, 1)

	q = search.NewQueryBuilder().
		AddBools(search.PlatformComponent, false).
		AddExactMatches(search.EntityType, storage.Alert_DEPLOYMENT.String()).
		ProtoQuery()
	results, err = s.searcher.Search(ctx, q, true)
	s.NoError(err)
	s.Len(results, 1)

	q = search.NewQueryBuilder().
		AddBools(search.PlatformComponent, true).
		ProtoQuery()
	results, err = s.searcher.Search(ctx, q, true)
	s.NoError(err)
	s.Len(results, 0)
}

func (s *AlertsSearchSuite) TestSearchResolved() {
	ids := []string{fixtureconsts.Alert1, fixtureconsts.Alert2, fixtureconsts.Alert3, fixtureconsts.Alert4}
	allAlertIds := make(map[string]bool)
	unresolvedAlertIds := make(map[string]bool)
	for i, id := range ids {
		alert := fixtures.GetAlert()
		alert.Id = id
		alert.EntityType = storage.Alert_DEPLOYMENT
		alert.PlatformComponent = false
		if i >= 2 {
			alert.State = storage.ViolationState_RESOLVED
		} else {
			unresolvedAlertIds[alert.Id] = true
		}
		allAlertIds[alert.Id] = true
		s.NoError(s.store.Upsert(ctx, alert))
		foundAlert, exists, err := s.store.Get(ctx, id)
		s.True(exists)
		s.NoError(err)
		protoassert.Equal(s.T(), alert, foundAlert)
	}
	results, err := s.searcher.Search(ctx, search.EmptyQuery(), false)
	s.NoError(err)
	// check that the result is in the allAlertIds map, then set the value to false, indicating it has already been found
	for _, result := range results {
		s.True(allAlertIds[result.ID])
		allAlertIds[result.ID] = false
	}
	// check that all ids were found
	for entry := range allAlertIds {
		s.False(allAlertIds[entry])
	}
	results, err = s.searcher.Search(ctx, search.EmptyQuery(), true)
	s.NoError(err)
	for _, result := range results {
		s.True(unresolvedAlertIds[result.ID])
		unresolvedAlertIds[result.ID] = false
	}
	for entry := range unresolvedAlertIds {
		s.False(unresolvedAlertIds[entry])
	}
}

func (s *AlertsSearchSuite) TestCountResolved() {
	ids := []string{fixtureconsts.Alert1, fixtureconsts.Alert2, fixtureconsts.Alert3, fixtureconsts.Alert4}
	for i, id := range ids {
		alert := fixtures.GetAlert()
		alert.Id = id
		alert.EntityType = storage.Alert_DEPLOYMENT
		alert.PlatformComponent = false
		if i >= 2 {
			alert.State = storage.ViolationState_RESOLVED
		}
		s.NoError(s.store.Upsert(ctx, alert))
		foundAlert, exists, err := s.store.Get(ctx, id)
		s.True(exists)
		s.NoError(err)
		protoassert.Equal(s.T(), alert, foundAlert)
	}
	results, err := s.searcher.Count(ctx, search.EmptyQuery(), false)
	s.NoError(err)
	s.Equal(4, results)

	results, err = s.searcher.Count(ctx, search.EmptyQuery(), true)
	s.NoError(err)
	s.Equal(2, results)
}
