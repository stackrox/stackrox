package m50tom51

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/rockshelper"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(integrationHealthMigrationTestSuite))
}

type integrationHealthMigrationTestSuite struct {
	suite.Suite

	rocksdb *rocksdb.RocksDB
	boltdb  *bolt.DB

	databases *dbTypes.Databases
}

func (suite *integrationHealthMigrationTestSuite) SetupTest() {
	boltdb, err := bolthelpers.NewTemp(testutils.DBFileName(suite))
	if err != nil {
		suite.FailNow("Failed to make BoltDB", err.Error())
	}
	suite.NoError(boltdb.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(imageIntegrationsBucketName); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(notifierBucketName); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(externalBackupsBucketName); err != nil {
			return err
		}
		return nil
	}))

	rocksDB, err := rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.rocksdb = rocksDB
	suite.boltdb = boltdb
	suite.databases = &dbTypes.Databases{BoltDB: boltdb, RocksDB: rocksDB.DB}
}

func (suite *integrationHealthMigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.databases.BoltDB)
	rocksdbtest.TearDownRocksDB(suite.rocksdb)
}

func (suite *integrationHealthMigrationTestSuite) TestDefaultIntegrationHealthMigration() {
	imageIntegration := &storage.ImageIntegration{
		Id:   "image1",
		Name: "Docker Hub Integration",
	}
	imageIntegrationBucket := bolthelpers.TopLevelRef(suite.databases.BoltDB, imageIntegrationsBucketName)

	err := imageIntegrationBucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(imageIntegration)
		if err != nil {
			return err
		}
		return b.Put([]byte(imageIntegration.Id), bytes)
	})
	suite.NoError(err)

	notifier := &storage.Notifier{
		Id:   "notifier1",
		Name: "Slack Notifier",
	}
	notifiersBucket := bolthelpers.TopLevelRef(suite.databases.BoltDB, notifierBucketName)

	err = notifiersBucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(notifier)
		if err != nil {
			return err
		}
		return b.Put([]byte(notifier.Id), bytes)
	})
	suite.NoError(err)

	backup := &storage.ExternalBackup{
		Id:   "backup1",
		Name: "S3 Backup",
	}
	backupsBucket := bolthelpers.TopLevelRef(suite.databases.BoltDB, externalBackupsBucketName)

	err = backupsBucket.Update(func(b *bolt.Bucket) error {
		bytes, err := proto.Marshal(backup)
		if err != nil {
			return err
		}
		return b.Put([]byte(backup.Id), bytes)
	})
	suite.NoError(err)

	err = insertDefaultHealthStatus(suite.boltdb, suite.rocksdb.DB)
	suite.NoError(err)

	suite.checkExists("image1", "health data for image integration incorrect")
	suite.checkExists("notifier1", "health data for notifier integration incorrect")
	suite.checkExists("backup1", "health data for external backup integration incorrect")

	suite.Assert().Equal(suite.checkNumEntries(), 3)
}

func (suite *integrationHealthMigrationTestSuite) checkNumEntries() int {
	readOpts := gorocksdb.NewDefaultReadOptions()
	it := suite.rocksdb.NewIterator(readOpts)
	defer it.Close()

	count := 0
	for it.Seek(integrationHealthPrefix); it.ValidForPrefix(integrationHealthPrefix); it.Next() {
		count++
	}
	return count
}

func (suite *integrationHealthMigrationTestSuite) checkExists(id string, errMsg string) {
	readOpts := gorocksdb.NewDefaultReadOptions()
	msg, exists, err := rockshelper.ReadFromRocksDB(suite.databases.RocksDB, readOpts, &storage.IntegrationHealth{}, integrationHealthPrefix, []byte(id))
	suite.NoError(err)
	suite.True(exists)
	suite.Equalf(msg.(*storage.IntegrationHealth).Id, id, errMsg)
}
