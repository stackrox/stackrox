//go:build sql_integration

package index

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/processindicator/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	fakeID = fixtures.GetProcessIndicator().GetId()
	ctx    = sac.WithAllAccess(context.Background())
)

func TestIndicatorIndex(t *testing.T) {
	suite.Run(t, new(IndicatorIndexTestSuite))
}

type IndicatorIndexTestSuite struct {
	suite.Suite

	db      *pgtest.TestPostgres
	indexer Indexer
}

func (suite *IndicatorIndexTestSuite) SetupSuite() {
	suite.db = pgtest.ForT(suite.T())
	suite.indexer = postgres.NewIndexer(suite.db)

	store := postgres.New(suite.db)

	process := fixtures.GetProcessIndicator()
	suite.NoError(store.Upsert(ctx, process))
}

func (suite *IndicatorIndexTestSuite) TestProcessIndicatorSearch() {
	mockIndicator := fixtures.GetProcessIndicator()
	processSignal := mockIndicator.GetSignal()

	cases := []struct {
		name        string
		q           *v1.Query
		expectedIDs []string
	}{
		{
			name:        "Empty",
			q:           search.EmptyQuery(),
			expectedIDs: []string{fakeID},
		},
		{
			name:        "Deployment id",
			q:           search.NewQueryBuilder().AddStrings(search.DeploymentID, mockIndicator.GetDeploymentId()).ProtoQuery(),
			expectedIDs: []string{fakeID},
		},
		{
			name:        "Matching exec path",
			q:           search.NewQueryBuilder().AddStrings(search.ProcessExecPath, processSignal.GetExecFilePath()).ProtoQuery(),
			expectedIDs: []string{fakeID},
		},
		{
			name:        "Matching name",
			q:           search.NewQueryBuilder().AddStrings(search.ProcessName, processSignal.GetName()).ProtoQuery(),
			expectedIDs: []string{fakeID},
		},
		{
			name:        "Matching command line 1st arg",
			q:           search.NewQueryBuilder().AddStrings(search.ProcessArguments, processSignal.GetArgs()).ProtoQuery(),
			expectedIDs: []string{fakeID},
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			results, err := suite.indexer.Search(ctx, c.q)
			require.NoError(t, err)
			resultIDs := make([]string, 0, len(results))
			for _, r := range results {
				resultIDs = append(resultIDs, r.ID)
			}
			assert.ElementsMatch(t, resultIDs, c.expectedIDs)
		})
	}
}

func (suite *IndicatorIndexTestSuite) TearDownSuite() {
	suite.db.Teardown(suite.T())
}
