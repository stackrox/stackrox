package m3to4

import (
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/bolthelpers/testhelpers"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestMigration3To4(t *testing.T) {
	suite.Run(t, new(Migration3To4TestSuite))
}

type Migration3To4TestSuite struct {
	suite.Suite

	db *bolt.DB
}

func (suite *Migration3To4TestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(clusterBucketName); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists(clusterStatusBucketName)
		return err
	}))
	suite.db = db
}

func (suite *Migration3To4TestSuite) TearDownTest() {
	testutils.TearDownDB(suite.db)
}

func getNormalClusterNoStatus(id string) *storage.Cluster {
	return &storage.Cluster{
		Id:        id,
		MainImage: "stackrox/main:2.3",
	}
}

func getNormalCluster(id string) *storage.Cluster {
	cluster := getNormalClusterNoStatus(id)
	cluster.Status = &storage.ClusterStatus{
		ProviderMetadata: &storage.ProviderMetadata{
			Region: "temp",
		},
		OrchestratorMetadata: &storage.OrchestratorMetadata{
			Version: "1.2.3",
		},
		SensorVersion: "2.3.13",
	}
	return cluster
}

func getLegacyCluster(id string) *storage.Cluster {
	cluster := getNormalCluster(id)
	if metadata := cluster.GetStatus().GetProviderMetadata(); metadata != nil {
		cluster.DEPRECATEDProviderMetadata = metadata
		cluster.Status.ProviderMetadata = nil
	}
	if metadata := cluster.GetStatus().GetOrchestratorMetadata(); metadata != nil {
		cluster.DEPRECATEDOrchestratorMetadata = metadata
		cluster.Status.OrchestratorMetadata = nil
	}
	return cluster
}

func (suite *Migration3To4TestSuite) mustGetCluster(id string) *storage.Cluster {
	cluster := testhelpers.MustGetObject(suite.T(), suite.db, clusterBucketName, id, func() proto.Message {
		return new(storage.Cluster)
	}).(*storage.Cluster)
	status := testhelpers.MustGetObject(suite.T(), suite.db, clusterStatusBucketName, id, func() proto.Message {
		return new(storage.ClusterStatus)
	})
	if status != nil {
		cluster.Status = status.(*storage.ClusterStatus)
	}
	return cluster
}

func (suite *Migration3To4TestSuite) mustInsertCluster(cluster *storage.Cluster) {
	testhelpers.MustInsertObject(suite.T(), suite.db, clusterBucketName, cluster.GetId(), cluster)
	if cluster.GetStatus() != nil {
		testhelpers.MustInsertObject(suite.T(), suite.db, clusterStatusBucketName, cluster.GetId(), cluster.GetStatus())
	}
}

func (suite *Migration3To4TestSuite) TestWithSimpleClusters() {
	ids := []string{"id1", "id2", "id3"}

	for _, id := range ids {
		suite.mustInsertCluster(getNormalClusterNoStatus(id))
	}

	suite.NoError(clusterStatusMigration.Run(suite.db, nil))

	for _, id := range ids {
		got := suite.mustGetCluster(id)
		suite.Equal(getNormalClusterNoStatus(id), got)
	}
}

func (suite *Migration3To4TestSuite) TestWithLegacyClusters() {
	ids := []string{"id1", "id2"}
	for _, id := range ids {
		suite.mustInsertCluster(getLegacyCluster(id))
	}

	suite.NoError(clusterStatusMigration.Run(suite.db, nil))

	for _, id := range ids {
		got := suite.mustGetCluster(id)
		suite.Equal(getNormalCluster(id), got)
	}
}

func (suite *Migration3To4TestSuite) TestNoClusters() {
	suite.NoError(clusterStatusMigration.Run(suite.db, nil))
}
