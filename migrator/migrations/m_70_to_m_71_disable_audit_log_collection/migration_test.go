package m70tom71

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
)

func TestDisableAuditLogCollectionMigration(t *testing.T) {
	suite.Run(t, new(disableAuditLogTestSuite))
}

type disableAuditLogTestSuite struct {
	suite.Suite

	db *rocksdb.RocksDB
}

func (suite *disableAuditLogTestSuite) SetupTest() {
	suite.db = rocksdbtest.RocksDBForT(suite.T())
}

func (suite *disableAuditLogTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *disableAuditLogTestSuite) TestMigrateClusters() {
	clusters := []*storage.Cluster{
		{
			Id:   "1",
			Type: storage.ClusterType_KUBERNETES_CLUSTER,
		},
		{
			Id:   "2",
			Type: storage.ClusterType_OPENSHIFT_CLUSTER,
		},
		{
			Id:   "3",
			Type: storage.ClusterType_OPENSHIFT4_CLUSTER,
			DynamicConfig: &storage.DynamicClusterConfig{
				DisableAuditLogs: false,
			},
		},
		{
			Id:   "4",
			Type: storage.ClusterType_OPENSHIFT4_CLUSTER,
			DynamicConfig: &storage.DynamicClusterConfig{
				DisableAuditLogs: true,
			},
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
	suite.NoError(disableAuditLogCollection(suite.db.DB))

	expected := []*storage.Cluster{
		{
			Id:   "1",
			Type: storage.ClusterType_KUBERNETES_CLUSTER,
			DynamicConfig: &storage.DynamicClusterConfig{
				DisableAuditLogs: true,
			},
		},
		{
			Id:   "2",
			Type: storage.ClusterType_OPENSHIFT_CLUSTER,
			DynamicConfig: &storage.DynamicClusterConfig{
				DisableAuditLogs: true,
			},
		},
		{
			Id:   "3",
			Type: storage.ClusterType_OPENSHIFT4_CLUSTER,
			DynamicConfig: &storage.DynamicClusterConfig{
				DisableAuditLogs: false,
			},
		},
		{
			Id:   "4",
			Type: storage.ClusterType_OPENSHIFT4_CLUSTER,
			DynamicConfig: &storage.DynamicClusterConfig{
				DisableAuditLogs: true,
			},
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
