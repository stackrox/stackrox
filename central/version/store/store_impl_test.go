package store

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
)

func TestVersionStore(t *testing.T) {
	suite.Run(t, new(VersionStoreTestSuite))
}

type VersionStoreTestSuite struct {
	suite.Suite

	boltDB  *bolt.DB
	rocksDB *rocksdb.RocksDB
	store   Store
}

func (suite *VersionStoreTestSuite) SetupTest() {
	boltDB, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	suite.Require().NoError(err, "Failed to make BoltDB")

	rocksDB := rocksdbtest.RocksDBForT(suite.T())

	suite.boltDB = boltDB
	suite.rocksDB = rocksDB
	suite.store = New(boltDB, rocksDB)
}

func (suite *VersionStoreTestSuite) TearDownTest() {
	suite.NoError(suite.boltDB.Close())
	suite.rocksDB.Close()
}

func (suite *VersionStoreTestSuite) TestVersionStore() {
	v, err := suite.store.GetVersion()
	suite.NoError(err)
	suite.Nil(v)

	for _, version := range []int32{2, 5, 19} {
		protoVersion := &storage.Version{SeqNum: version, Version: fmt.Sprintf("Version %d", version)}
		suite.NoError(suite.store.UpdateVersion(protoVersion))
		got, err := suite.store.GetVersion()
		suite.NoError(err)
		suite.Equal(protoVersion, got)
	}
}

func (suite *VersionStoreTestSuite) TestVersionMismatch() {
	boltVersion := &storage.Version{SeqNum: 2, Version: "Version 2"}
	boltVersionBytes, err := boltVersion.Marshal()
	suite.Require().NoError(err)

	rocksVersion := &storage.Version{SeqNum: 3, Version: "Version 3"}
	rocksVersionBytes, err := rocksVersion.Marshal()
	suite.Require().NoError(err)

	suite.NoError(suite.rocksDB.Put(gorocksdb.NewDefaultWriteOptions(), versionBucket, rocksVersionBytes))

	suite.NoError(suite.boltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(versionBucket)
		return bucket.Put(key, boltVersionBytes)
	}))

	_, err = suite.store.GetVersion()
	suite.Error(err)
}
