package store

import (
	"fmt"
	"testing"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stretchr/testify/suite"
)

func TestVersionStore(t *testing.T) {
	suite.Run(t, new(VersionStoreTestSuite))
}

type VersionStoreTestSuite struct {
	suite.Suite

	boltDB   *bolt.DB
	badgerDB *badger.DB

	store Store
}

func (suite *VersionStoreTestSuite) SetupTest() {
	boltDB, err := bolthelper.NewTemp(suite.T().Name() + ".db")
	suite.Require().NoError(err, "Failed to make BoltDB")

	badgerDB, _, err := badgerhelper.NewTemp(suite.T().Name())
	suite.Require().NoError(err, "failed to create badger DB")

	suite.boltDB = boltDB
	suite.badgerDB = badgerDB
	suite.store = New(boltDB, badgerDB)
}

func (suite *VersionStoreTestSuite) TearDownTest() {
	suite.NoError(suite.boltDB.Close())
	suite.NoError(suite.badgerDB.Close())
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

	badgerVersion := &storage.Version{SeqNum: 3, Version: "Version 3"}
	badgerVersionBytes, err := badgerVersion.Marshal()
	suite.Require().NoError(err)

	suite.NoError(suite.badgerDB.Update(func(txn *badger.Txn) error {
		return txn.Set(versionBucket, badgerVersionBytes)
	}))
	suite.NoError(suite.boltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(versionBucket)
		return bucket.Put(key, boltVersionBytes)
	}))

	_, err = suite.store.GetVersion()
	suite.Error(err)
}
