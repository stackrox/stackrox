package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/alert/convert"
	"github.com/stackrox/rox/central/alert/index/mappings"
	"github.com/stackrox/rox/central/globalindex"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
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

func (suite *alertIndexTestSuite) SetupTest() {
	tmpIndex, err := globalindex.TempInitializeIndices("")
	suite.Require().NoError(err)

	suite.bleveIndex = tmpIndex
	suite.indexer = New(tmpIndex)
}

func (suite *alertIndexTestSuite) TearDownTest() {
	suite.NoError(suite.bleveIndex.Close())
}

func (suite *alertIndexTestSuite) TestDefaultStaleness() {
	const nonStaleID = "NONSTALE"
	const staleID = "STALE"

	suite.NoError(suite.indexer.AddListAlert(convert.AlertToListAlert(fixtures.GetAlertWithID(nonStaleID))))
	staleAlert := fixtures.GetAlertWithID(staleID)
	staleAlert.State = storage.ViolationState_RESOLVED
	suite.NoError(suite.indexer.AddListAlert(convert.AlertToListAlert(staleAlert)))

	var cases = []struct {
		name             string
		state            string
		expectedAlertIDs []string
	}{
		{
			name:             "no stale field",
			expectedAlertIDs: []string{nonStaleID},
		},
		{
			name:             "state = active",
			state:            storage.ViolationState_ACTIVE.String(),
			expectedAlertIDs: []string{nonStaleID},
		},
		{
			name:             "state = stale",
			state:            storage.ViolationState_RESOLVED.String(),
			expectedAlertIDs: []string{staleID},
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			qb := search.NewQueryBuilder()
			if c.state != "" {
				qb.AddStrings(search.ViolationState, c.state)
			}
			alerts, err := suite.indexer.Search(qb.ProtoQuery())
			assert.NoError(t, err)

			alertIDs := make([]string, 0, len(alerts))
			for _, alert := range alerts {
				alertIDs = append(alertIDs, alert.ID)
			}

			assert.ElementsMatch(t, alertIDs, c.expectedAlertIDs)
		})
	}
}

// This test also tests xref because the Severity enum is buried inside Policy
func (suite *alertIndexTestSuite) TestEnums() {
	suite.NoError(suite.indexer.AddListAlert(convert.AlertToListAlert(fixtures.GetAlert())))

	var cases = []struct {
		name             string
		query            *v1.Query
		expectedAlertIDs []string
	}{
		{
			name:             "match severity",
			query:            search.NewQueryBuilder().AddStrings(search.Severity, "low").ProtoQuery(),
			expectedAlertIDs: []string{fixtures.GetAlert().GetId()},
		},
		{
			name:             "no match severity",
			query:            search.NewQueryBuilder().AddStrings(search.Severity, "high").ProtoQuery(),
			expectedAlertIDs: []string{},
		},
	}

	for _, c := range cases {
		suite.T().Run(c.name, func(t *testing.T) {
			alerts, err := suite.indexer.Search(c.query)
			assert.NoError(t, err)

			alertIDs := make([]string, 0, len(alerts))
			for _, alert := range alerts {
				alertIDs = append(alertIDs, alert.ID)
			}

			assert.ElementsMatch(t, alertIDs, c.expectedAlertIDs)
		})
	}
}

// This test ensures that the search options for both list alerts and alerts are identical. This is necessary
// because we load and index list alerts from the DB for performance
func TestListAlertAndAlertWalkAreEqual(t *testing.T) {
	listOptions := blevesearch.Walk(v1.SearchCategory_ALERTS, "alert", (*storage.ListAlert)(nil))
	assert.Equal(t, listOptions, mappings.OptionsMap)
}
