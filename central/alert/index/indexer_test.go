package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestAlertIndex(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(alertIndexTestSuite))
}

type alertIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index
	indexer    Indexer
}

func (suite *alertIndexTestSuite) SetupSuite() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.indexer = New(tmpIndex)

}

func (suite *alertIndexTestSuite) TeardownSuite() {
	suite.bleveIndex.Close()
}

func (suite *alertIndexTestSuite) TestDefaultStaleness() {
	const nonStaleID = "NONSTALE"
	const staleID = "STALE"

	suite.NoError(suite.indexer.AddAlert(fixtures.GetAlertWithID(nonStaleID)))
	staleAlert := fixtures.GetAlertWithID(staleID)
	staleAlert.Stale = true
	suite.NoError(suite.indexer.AddAlert(staleAlert))

	var cases = []struct {
		name             string
		staleValue       string
		expectedAlertIDs []string
	}{
		{
			name:             "no stale field",
			expectedAlertIDs: []string{nonStaleID},
		},
		{
			name:             "stale = false",
			staleValue:       "false",
			expectedAlertIDs: []string{nonStaleID},
		},
		{
			name:             "stale = true",
			staleValue:       "true",
			expectedAlertIDs: []string{staleID},
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			qb := search.NewQueryBuilder()
			if c.staleValue != "" {
				qb.AddStrings(search.Stale, c.staleValue)
			}
			alerts, err := suite.indexer.SearchAlerts(qb.ProtoQuery())
			assert.NoError(t, err)

			alertIDs := make([]string, 0, len(alerts))
			for _, alert := range alerts {
				alertIDs = append(alertIDs, alert.ID)
			}

			assert.ElementsMatch(t, alertIDs, c.expectedAlertIDs)
		})
	}
}
