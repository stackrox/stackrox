package index

import (
	"testing"

	"github.com/blevesearch/bleve"
	"github.com/stackrox/stackrox/central/globalindex"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stretchr/testify/suite"
)

func TestClusterIndex(t *testing.T) {
	suite.Run(t, new(ClusterIndexTestSuite))
}

type ClusterIndexTestSuite struct {
	suite.Suite

	bleveIndex bleve.Index
	indexer    Indexer
}

func (suite *ClusterIndexTestSuite) SetupTest() {
	var err error
	suite.bleveIndex, err = globalindex.MemOnlyIndex()
	suite.Require().NoError(err)

	suite.indexer = New(suite.bleveIndex)
}

func (suite *ClusterIndexTestSuite) TearDownTest() {
	suite.NoError(suite.bleveIndex.Close())
}

func (suite *ClusterIndexTestSuite) TestIndexing() {
	cluster := &storage.Cluster{
		Id:   "cluster",
		Name: "cluster1",
	}

	suite.NoError(suite.indexer.AddCluster(cluster))

	q := search.NewQueryBuilder().AddStrings(search.Cluster, "cluster1").ProtoQuery()
	results, err := suite.indexer.Search(q)
	suite.NoError(err)
	suite.Len(results, 1)
}
