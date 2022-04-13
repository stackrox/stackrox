package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/stackrox/central/globalindex"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/fixtures"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var fakeID = fixtures.GetProcessIndicator().GetId()

func TestIndicatorIndex(t *testing.T) {
	suite.Run(t, new(IndicatorIndexTestSuite))
}

type IndicatorIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index

	indexer Indexer
}

func (suite *IndicatorIndexTestSuite) SetupSuite() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.indexer = New(tmpIndex)

	process := fixtures.GetProcessIndicator()
	suite.NoError(suite.indexer.AddProcessIndicator(process))
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
			results, err := suite.indexer.Search(c.q)
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
	suite.NoError(suite.bleveIndex.Close())
}
