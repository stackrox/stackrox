package index

import (
	"testing"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
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

func (suite *ClusterIndexTestSuite) TestSearchByLastContact() {
	timeNowMinusTwoDays, err := types.TimestampProto(time.Now().AddDate(0, 0, -2))
	suite.Require().NoError(err)

	timeNowMinusThreeDays, err := types.TimestampProto(time.Now().AddDate(0, 0, -3))
	suite.Require().NoError(err)

	timeNowMinusFourDays, err := types.TimestampProto(time.Now().AddDate(0, 0, -4))
	suite.Require().NoError(err)

	clusters := []*storage.Cluster{
		{
			Id: "Cluster_NoHealthStatus",
		},
		{
			Id: "Cluster_HealthStatus_UNAVAILABLE",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
			},
		},
		{
			Id: "Cluster_HealthStatus_HEALTHY",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY,
				LastContact:        types.TimestampNow(),
			},
		},
		{
			Id: "Cluster_HealthStatus_UNHEALTHY_since_2_days",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:        timeNowMinusTwoDays,
			},
		},
		{
			Id: "Cluster_HealthStatus_UNHEALTHY_since_3_days",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:        timeNowMinusThreeDays,
			},
		},
		{
			Id: "Cluster_HealthStatus_UNHEALTHY_since_4_days",
			HealthStatus: &storage.ClusterHealthStatus{
				SensorHealthStatus: storage.ClusterHealthStatus_UNHEALTHY,
				LastContact:        timeNowMinusFourDays,
			},
		},
	}

	expectedIDs := []string{
		"Cluster_HealthStatus_UNHEALTHY_since_3_days",
		"Cluster_HealthStatus_UNHEALTHY_since_4_days",
	}

	suite.NoError(suite.indexer.AddClusters(clusters))

	q := search.NewQueryBuilder().AddDays(search.LastContactTime, int64(3)).ProtoQuery()

	results, err := suite.indexer.Search(q)
	suite.NoError(err)

	actualIDs := search.ResultsToIDs(results)
	suite.Equal(expectedIDs, actualIDs)
}
