package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/rox/central/globalindex"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestProcessBaselineIndex(t *testing.T) {
	suite.Run(t, new(ProcessBaselineIndexTestSuite))
}

type ProcessBaselineIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index
	indexer    Indexer
}

func (suite *ProcessBaselineIndexTestSuite) SetupTest() {
	var err error
	suite.bleveIndex, err = globalindex.MemOnlyIndex()
	suite.Require().NoError(err)

	suite.indexer = New(suite.bleveIndex)
}

func (suite *ProcessBaselineIndexTestSuite) TearDownTest() {
	suite.NoError(suite.bleveIndex.Close())
}

func (suite *ProcessBaselineIndexTestSuite) getAndStoreBaseline() *storage.ProcessBaseline {
	baseline := fixtures.GetProcessBaselineWithID()
	suite.NotNil(baseline.GetElements())
	suite.Len(baseline.GetElements(), 1, "The fixture should return a baseline with exactly one process in it or this test should change")
	err := suite.indexer.AddProcessBaseline(baseline)
	suite.NoError(err)
	return baseline
}

func (suite *ProcessBaselineIndexTestSuite) search(q *v1.Query, expectedResultSize int) ([]search.Result, error) {
	results, err := suite.indexer.Search(q)
	suite.NoError(err)
	suite.Equal(expectedResultSize, len(results))
	return results, err
}

func (suite *ProcessBaselineIndexTestSuite) TestNoResults() {
	q := search.NewQueryBuilder().AddStringsHighlighted(search.DeploymentID, "This ID doesn't exist").ProtoQuery()
	_, err := suite.search(q, 0)
	suite.NoError(err)
	suite.getAndStoreBaseline()
	_, err = suite.search(q, 0)
	suite.NoError(err)
}

func (suite *ProcessBaselineIndexTestSuite) TestEmptySearch() {
	baseline1 := suite.getAndStoreBaseline()
	baseline2 := suite.getAndStoreBaseline()
	expectedSet := map[string]struct{}{
		baseline1.GetId(): {},
		baseline2.GetId(): {},
	}
	q := search.EmptyQuery()
	results, _ := suite.search(q, 2)
	resultSet := map[string]struct{}{}
	for _, r := range results {
		resultSet[r.ID] = struct{}{}
	}
	suite.Equal(expectedSet, resultSet)
}

func (suite *ProcessBaselineIndexTestSuite) TestAddSearchDeleteBaseline() {
	baseline := suite.getAndStoreBaseline()
	suite.getAndStoreBaseline() // Don't find this one

	q := search.NewQueryBuilder().AddStrings(search.DeploymentID, baseline.GetKey().GetDeploymentId()).ProtoQuery()
	results, err := suite.search(q, 1)
	suite.NoError(err)
	suite.Equal(baseline.GetId(), results[0].ID)

	err = suite.indexer.DeleteProcessBaseline(baseline.GetId())
	suite.NoError(err)
	results, err = suite.indexer.Search(q)
	suite.NoError(err)
	suite.Equal(0, len(results))
}

func (suite *ProcessBaselineIndexTestSuite) TestSearchByDeploymentID() {
	baseline := suite.getAndStoreBaseline()
	suite.getAndStoreBaseline() // Don't find this one

	q := search.NewQueryBuilder().AddStrings(search.DeploymentID, baseline.GetKey().GetDeploymentId()).ProtoQuery()
	results, err := suite.search(q, 1)
	suite.NoError(err)
	suite.Equal(baseline.GetId(), results[0].ID)
}
