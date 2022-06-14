package m64to65

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
)

func TestExecWebhookMigration(t *testing.T) {
	suite.Run(t, new(openshift4ClusterTypeMigrationTestSuite))
}

type openshift4ClusterTypeMigrationTestSuite struct {
	suite.Suite

	db *rocksdb.RocksDB
}

func (suite *openshift4ClusterTypeMigrationTestSuite) SetupTest() {
	suite.db = rocksdbtest.RocksDBForT(suite.T())
}

func (suite *openshift4ClusterTypeMigrationTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *openshift4ClusterTypeMigrationTestSuite) TestMigrateClustersWithExecWebhooks() {
	clusters := []*storage.Cluster{
		{
			Id:   "1",
			Type: storage.ClusterType_OPENSHIFT_CLUSTER,
		},
		{
			Id:                        "2",
			Type:                      storage.ClusterType_OPENSHIFT_CLUSTER,
			AdmissionControllerEvents: true,
		},
		{
			Id:   "3",
			Type: storage.ClusterType_KUBERNETES_CLUSTER,
		},
		{
			Id:                        "4",
			Type:                      storage.ClusterType_KUBERNETES_CLUSTER,
			AdmissionControllerEvents: true,
		},
	}

	wb := gorocksdb.NewWriteBatch()
	for _, c := range clusters {
		bytes, err := proto.Marshal(c)
		suite.NoError(err)

		wb.Put(rocksdbmigration.GetPrefixedKey(clustersPrefix, []byte(c.Id)), bytes)
	}
	err := suite.db.Write(gorocksdb.NewDefaultWriteOptions(), wb)
	suite.NoError(err)

	// Migrate the data
	suite.NoError(migrateOpenShiftClusterType(suite.db.DB))

	expected := []*storage.Cluster{
		{
			Id:                        "1",
			Type:                      storage.ClusterType_OPENSHIFT_CLUSTER,
			AdmissionControllerEvents: false,
		},
		{
			Id:                        "2",
			Type:                      storage.ClusterType_OPENSHIFT4_CLUSTER,
			AdmissionControllerEvents: true,
		},
		{
			Id:   "3",
			Type: storage.ClusterType_KUBERNETES_CLUSTER,
		},
		{
			Id:                        "4",
			Type:                      storage.ClusterType_KUBERNETES_CLUSTER,
			AdmissionControllerEvents: true,
		},
	}
	readOpts := gorocksdb.NewDefaultReadOptions()
	it := suite.db.NewIterator(readOpts)
	defer it.Close()

	migratedClusters := make([]*storage.Cluster, 0, len(expected))
	prefix := getPrefix()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		cluster := &storage.Cluster{}
		if err := proto.Unmarshal(it.Value().Data(), cluster); err != nil {
			suite.NoError(err)
		}
		migratedClusters = append(migratedClusters, cluster)
	}

	assert.ElementsMatch(suite.T(), expected, migratedClusters)
}
