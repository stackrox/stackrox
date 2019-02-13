package version

import (
	"testing"

	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/version/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/suite"
)

func TestEnsurer(t *testing.T) {
	suite.Run(t, new(EnsurerTestSuite))
}

type EnsurerTestSuite struct {
	suite.Suite

	boltDB       *bolt.DB
	badgerDB     *badger.DB
	versionStore store.Store
}

func (suite *EnsurerTestSuite) SetupTest() {
	boltDB, err := bolthelper.NewTemp(testutils.DBFileName(suite.Suite))
	suite.Require().NoError(err, "Failed to make BoltDB")

	badgerDB, _, err := badgerhelper.NewTemp(suite.T().Name())
	suite.Require().NoError(err, "Failed to create BadgerDB")

	suite.boltDB = boltDB
	suite.badgerDB = badgerDB
	suite.versionStore = store.New(boltDB, badgerDB)
}

func (suite *EnsurerTestSuite) TearDownTest() {
	suite.NoError(suite.boltDB.Close())
}

func (suite *EnsurerTestSuite) TestWithEmptyDB() {
	suite.NoError(Ensure(suite.boltDB, suite.badgerDB))
	version, err := suite.versionStore.GetVersion()
	suite.NoError(err)
	suite.Equal(migrations.CurrentDBVersionSeqNum, int(version.GetSeqNum()))
}

func (suite *EnsurerTestSuite) TestWithCurrentVersion() {
	suite.NoError(suite.versionStore.UpdateVersion(&storage.Version{SeqNum: migrations.CurrentDBVersionSeqNum}))
	suite.NoError(Ensure(suite.boltDB, suite.badgerDB))

	version, err := suite.versionStore.GetVersion()
	suite.NoError(err)
	suite.Equal(migrations.CurrentDBVersionSeqNum, int(version.GetSeqNum()))
}

func (suite *EnsurerTestSuite) TestWithIncorrectVersion() {
	suite.NoError(suite.versionStore.UpdateVersion(&storage.Version{SeqNum: migrations.CurrentDBVersionSeqNum - 2}))
	suite.Error(Ensure(suite.boltDB, suite.badgerDB))
}
