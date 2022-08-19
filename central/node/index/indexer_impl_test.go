package index

import (
	"testing"

	"github.com/blevesearch/bleve/v2"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestNodeIndex(t *testing.T) {
	suite.Run(t, new(NodeIndexTestSuite))
}

type NodeIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index
	indexer    Indexer
}

func (suite *NodeIndexTestSuite) SetupTest() {
	var err error
	suite.bleveIndex, err = globalindex.MemOnlyIndex()
	suite.Require().NoError(err)

	suite.indexer = New(suite.bleveIndex)
}

func (suite *NodeIndexTestSuite) TearDownTest() {
	suite.NoError(suite.bleveIndex.Close())
}

func (suite *NodeIndexTestSuite) TestIndexing() {
	node := &storage.Node{
		Id:   "nodeid",
		Name: "node1",
	}

	suite.NoError(suite.indexer.AddNode(node))

	q := search.NewQueryBuilder().AddStrings(search.Node, "node").ProtoQuery()
	results, err := suite.indexer.Search(q)
	suite.NoError(err)
	suite.Len(results, 1)
}
