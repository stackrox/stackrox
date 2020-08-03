package m44tom45

import (
	"os"
	"reflect"
	"testing"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(clusterRocksDBMigrationTestSuite))
}

type clusterRocksDBMigrationTestSuite struct {
	suite.Suite

	databases *dbTypes.Databases
	dir       string
}

func (suite *clusterRocksDBMigrationTestSuite) SetupTest() {
	db, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(clusterBucketName); err != nil {
			return err
		}

		if _, err := tx.CreateBucketIfNotExists(clusterStatusBucketName); err != nil {
			return err
		}

		_, err := tx.CreateBucketIfNotExists(clusterLastContactTimeBucketName)
		return err
	}))

	rocksDB, dir, err := rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.databases = &dbTypes.Databases{BoltDB: db, RocksDB: rocksDB.DB}
	suite.dir = dir
}

func (suite *clusterRocksDBMigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.databases.BoltDB)
	_ = os.RemoveAll(suite.dir)
}

func insertMessage(bucket bolthelpers.BucketRef, id string, pb proto.Message) error {
	if pb == nil {
		return nil
	}
	if reflect.TypeOf(pb).Kind() == reflect.Ptr && reflect.ValueOf(pb).IsNil() {
		return nil
	}
	return bucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		return b.Put([]byte(id), bytes)
	})
}

func (suite *clusterRocksDBMigrationTestSuite) TestClusterRocksDBMigration() {
	ts := types.TimestampNow()
	cases := []struct {
		boltCluster        *storage.Cluster
		boltStatus         *storage.ClusterStatus
		boltLastContact    *types.Timestamp
		rocksDBCluster     *storage.Cluster
		rocksDBHealth      *storage.ClusterHealthStatus
		healthShouldExists bool
	}{
		{
			boltCluster: &storage.Cluster{
				Id:   "1",
				Name: "cluster1",
			},
			boltStatus: &storage.ClusterStatus{
				SensorVersion: "s1",
			},
			boltLastContact: ts,
			rocksDBCluster: &storage.Cluster{
				Id:   "1",
				Name: "cluster1",
				Status: &storage.ClusterStatus{
					SensorVersion: "s1",
				},
			},
			rocksDBHealth: &storage.ClusterHealthStatus{
				LastUpdated: ts,
			},
			healthShouldExists: true,
		},
		{
			boltCluster: &storage.Cluster{
				Id:   "2",
				Name: "cluster2",
			},
			boltStatus: &storage.ClusterStatus{
				SensorVersion: "s2",
			},
			boltLastContact: ts,
			rocksDBCluster: &storage.Cluster{
				Id:   "2",
				Name: "cluster2",
				Status: &storage.ClusterStatus{
					SensorVersion: "s2",
				},
			},
			rocksDBHealth: &storage.ClusterHealthStatus{
				LastUpdated: ts,
			},
			healthShouldExists: true,
		},
		{
			boltCluster: &storage.Cluster{
				Id:   "3",
				Name: "cluster3",
			},
			boltStatus: &storage.ClusterStatus{
				SensorVersion: "s3",
			},
			boltLastContact: ts,
			rocksDBCluster: &storage.Cluster{
				Id:   "3",
				Name: "cluster3",
				Status: &storage.ClusterStatus{
					SensorVersion: "s3",
				},
			},
			rocksDBHealth: &storage.ClusterHealthStatus{
				LastUpdated: ts,
			},
			healthShouldExists: true,
		},
		{
			boltCluster: &storage.Cluster{
				Id:   "4",
				Name: "cluster4",
			},
			rocksDBCluster: &storage.Cluster{
				Id:   "4",
				Name: "cluster4",
			},
			healthShouldExists: false,
		},
	}

	clusterBucket := bolthelpers.TopLevelRef(suite.databases.BoltDB, clusterBucketName)
	clusterStatusBucket := bolthelpers.TopLevelRef(suite.databases.BoltDB, clusterStatusBucketName)
	clusterLastContactBucket := bolthelpers.TopLevelRef(suite.databases.BoltDB, clusterLastContactTimeBucketName)

	for _, c := range cases {
		suite.NoError(insertMessage(clusterBucket, c.boltCluster.GetId(), c.boltCluster))
		suite.NoError(insertMessage(clusterStatusBucket, c.boltCluster.GetId(), c.boltStatus))
		suite.NoError(insertMessage(clusterLastContactBucket, c.boltCluster.GetId(), c.boltLastContact))
	}

	suite.NoError(migrateClusterBuckets(suite.databases))

	readOpts := gorocksdb.NewDefaultReadOptions()
	for _, c := range cases {
		cluster, exists, err := getFromRocksDB(suite.databases.RocksDB, readOpts, &storage.Cluster{}, clusterBucketName, []byte(c.boltCluster.GetId()))
		suite.NoError(err)
		suite.True(exists)
		suite.EqualValues(c.rocksDBCluster, cluster.(*storage.Cluster))
	}

	for _, c := range cases {
		health, exists, err := getFromRocksDB(suite.databases.RocksDB, readOpts, &storage.ClusterHealthStatus{}, clusterHealthStatusBucketName, []byte(c.boltCluster.GetId()))
		suite.NoError(err)
		suite.Equal(c.healthShouldExists, exists)
		if exists {
			suite.EqualValues(c.rocksDBHealth, health.(*storage.ClusterHealthStatus))
		}
	}
}

func getFromRocksDB(db *gorocksdb.DB, opts *gorocksdb.ReadOptions, msg proto.Message, prefix []byte, id []byte) (proto.Message, bool, error) {
	key := rocksdbmigration.GetPrefixedKey(prefix, id)
	slice, err := db.Get(opts, key)
	if err != nil {
		return nil, false, errors.Wrapf(err, "getting key %s", key)
	}
	defer slice.Free()
	if !slice.Exists() {
		return nil, false, nil
	}
	if err := proto.Unmarshal(slice.Data(), msg); err != nil {
		return nil, false, errors.Wrapf(err, "deserializing object with key %s", key)
	}
	return msg, true, nil
}
