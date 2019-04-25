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

func TestProcessWhitelistIndex(t *testing.T) {
	suite.Run(t, new(ProcessWhitelistIndexTestSuite))
}

type ProcessWhitelistIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index
	indexer    Indexer
}

func (suite *ProcessWhitelistIndexTestSuite) SetupTest() {
	var err error
	suite.bleveIndex, err = globalindex.MemOnlyIndex()
	suite.Require().NoError(err)

	suite.indexer = New(suite.bleveIndex)
}

func (suite *ProcessWhitelistIndexTestSuite) TearDownTest() {
	suite.NoError(suite.bleveIndex.Close())
}

func (suite *ProcessWhitelistIndexTestSuite) getAndStoreWhitelist() *storage.ProcessWhitelist {
	whitelist := fixtures.GetProcessWhitelistWithID()
	suite.NotNil(whitelist.GetElements())
	suite.Len(whitelist.GetElements(), 1, "The fixture should return a whitelist with exactly one process in it or this test should change")
	err := suite.indexer.AddWhitelist(whitelist)
	suite.NoError(err)
	return whitelist
}

func (suite *ProcessWhitelistIndexTestSuite) search(q *v1.Query, expectedResultSize int) ([]search.Result, error) {
	results, err := suite.indexer.Search(q)
	suite.NoError(err)
	suite.Equal(expectedResultSize, len(results))
	return results, err
}

func (suite *ProcessWhitelistIndexTestSuite) TestNoResults() {
	q := search.NewQueryBuilder().AddStringsHighlighted(search.ProcessName, "This ID doesn't exist").ProtoQuery()
	_, err := suite.search(q, 0)
	suite.NoError(err)
	suite.getAndStoreWhitelist()
	_, err = suite.search(q, 0)
	suite.NoError(err)
}

func (suite *ProcessWhitelistIndexTestSuite) TestEmptySearch() {
	whitelist1 := suite.getAndStoreWhitelist()
	whitelist2 := suite.getAndStoreWhitelist()
	expectedSet := map[string]struct{}{
		whitelist1.GetId(): {},
		whitelist2.GetId(): {},
	}
	q := search.EmptyQuery()
	results, _ := suite.search(q, 2)
	resultSet := map[string]struct{}{}
	for _, r := range results {
		resultSet[r.ID] = struct{}{}
	}
	suite.Equal(expectedSet, resultSet)
}

func (suite *ProcessWhitelistIndexTestSuite) TestAddSearchDeleteWhitelist() {
	whitelist := suite.getAndStoreWhitelist()
	suite.getAndStoreWhitelist() // Don't find this one

	q := search.NewQueryBuilder().AddStrings(search.ProcessName, whitelist.Elements[0].GetProcessName()).ProtoQuery()
	results, err := suite.search(q, 1)
	suite.NoError(err)
	suite.Equal(whitelist.GetId(), results[0].ID)

	err = suite.indexer.DeleteWhitelist(whitelist.GetId())
	suite.NoError(err)
	results, err = suite.indexer.Search(q)
	suite.NoError(err)
	suite.Equal(0, len(results))
}

func (suite *ProcessWhitelistIndexTestSuite) TestSearchByDeploymentID() {
	whitelist := suite.getAndStoreWhitelist()
	suite.getAndStoreWhitelist() // Don't find this one

	q := search.NewQueryBuilder().AddStrings(search.DeploymentID, whitelist.GetKey().GetDeploymentId()).ProtoQuery()
	results, err := suite.search(q, 1)
	suite.NoError(err)
	suite.Equal(whitelist.GetId(), results[0].ID)
}
