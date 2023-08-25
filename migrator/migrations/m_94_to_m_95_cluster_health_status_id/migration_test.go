package m94tom95

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

var (
	clusterBucket = []byte("cluster")
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(clusterHealthStatusIDTestSuite))
}

type clusterHealthStatusIDTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (s *clusterHealthStatusIDTestSuite) SetupTest() {
	rocksDB, err := rocksdb.NewTemp(s.T().Name())
	s.NoError(err)

	s.db = rocksDB
	s.databases = &dbTypes.Databases{RocksDB: rocksDB.DB}
}

func (s *clusterHealthStatusIDTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(s.db)
}

func (s *clusterHealthStatusIDTestSuite) TestMigrationAddsIdToClusterHealthStatus() {
	existingCluster := &storage.Cluster{
		Id:                 uuid.NewV4().String(),
		Name:               "Fake cluster 1",
		MainImage:          "docker.io/stackrox/rox:latest",
		CentralApiEndpoint: "central.stackrox:443",
	}

	key := rocksdbmigration.GetPrefixedKey(clusterBucket, []byte(existingCluster.GetId()))
	value, err := proto.Marshal(existingCluster)
	s.NoError(err)
	s.NoError(s.databases.RocksDB.Put(writeOpts, key, value))

	// Add in a cluster health status just to validate that it won't get picked up by the iterator
	chs := &storage.ClusterHealthStatus{SensorHealthStatus: storage.ClusterHealthStatus_HEALTHY}
	chsKey := rocksdbmigration.GetPrefixedKey([]byte("clusters_health_status"), []byte(existingCluster.GetId()))
	chsValue, err := proto.Marshal(chs)
	s.NoError(err)
	s.NoError(s.databases.RocksDB.Put(writeOpts, chsKey, chsValue))

	err = addIDToClusterHealthStatus(s.databases)
	s.NoError(err)

	s.validateIDAdded(clusterHealthStatusBucket)
}

func (s *clusterHealthStatusIDTestSuite) validateIDAdded(_ []byte) {
	it := s.databases.RocksDB.NewIterator(readOpts)
	defer it.Close()

	for it.Seek(clusterHealthStatusBucket); it.ValidForPrefix(clusterHealthStatusBucket); it.Next() {
		healthStatus := &storage.ClusterHealthStatus{}
		s.NoError(proto.Unmarshal(it.Value().Data(), healthStatus))
		s.NotEmpty(healthStatus.GetId(), "expected health status id to be populated, but it is not")
	}
}
