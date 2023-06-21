//go:build sql_integration

package postgres

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

var (
	ctx = sac.WithAllAccess(context.Background())
)

type AlertsIndexSuite struct {
	suite.Suite

	pool    postgres.DB
	store   Store
	indexer *indexerImpl
}

func TestAlertsIndex(t *testing.T) {
	suite.Run(t, new(AlertsIndexSuite))
}

func (s *AlertsIndexSuite) SetupTest() {
	source := pgtest.GetConnectionString(s.T())
	config, err := postgres.ParseConfig(source)
	s.Require().NoError(err)
	s.pool, err = postgres.New(context.Background(), config)
	s.Require().NoError(err)

	Destroy(ctx, s.pool)
	gormDB := pgtest.OpenGormDB(s.T(), source)
	defer pgtest.CloseGormDB(s.T(), gormDB)
	s.store = CreateTableAndNewStore(ctx, s.pool, gormDB)
	s.indexer = NewIndexer(s.pool)
}

func (s *AlertsIndexSuite) TearDownTest() {
	s.pool.Close()
}

func (s *AlertsIndexSuite) TestIndex() {

	alert := fixtures.GetAlert()
	foundAlert, exists, err := s.store.Get(ctx, alert.GetId())
	s.NoError(err)
	s.False(exists)
	s.Nil(foundAlert)

	s.NoError(s.store.Upsert(ctx, alert))
	foundAlert, exists, err = s.store.Get(ctx, alert.GetId())
	s.NoError(err)
	s.True(exists)
	s.Equal(alert, foundAlert)

	// Common alert searches
	results, err := s.indexer.Search(ctx, search.NewQueryBuilder().AddExactMatches(search.DeploymentID, alert.GetDeployment().GetId()).ProtoQuery())
	s.NoError(err)
	s.Len(results, 1)

	q := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, alert.GetDeployment().GetId()).
		AddExactMatches(search.PolicyID, alert.GetPolicy().GetId()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		ProtoQuery()
	results, err = s.indexer.Search(ctx, q)
	s.NoError(err)
	s.Len(results, 1)
}
