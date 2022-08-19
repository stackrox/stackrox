package m107tom108

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"go.etcd.io/bbolt"
)

var (
	unmigratedPSs = []*storage.PermissionSet{
		{
			Id:   "id0",
			Name: "ps0",
			ResourceToAccess: map[string]storage.Access{
				"AuthPlugin": storage.Access_READ_ACCESS,
				"Image":      storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:   "id1",
			Name: "ps1",
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	unmigratedPSsAfterMigration = []*storage.PermissionSet{
		{
			Id:   "id0",
			Name: "ps0",
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:   "id1",
			Name: "ps1",
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	alreadyMigratedPSs = []*storage.PermissionSet{
		{
			Id:               "id2",
			Name:             "ps2",
			ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_WRITE_ACCESS},
		},
		{
			Id:               "id3",
			Name:             "ps3",
			ResourceToAccess: map[string]storage.Access{"Image": storage.Access_READ_WRITE_ACCESS},
		},
	}
)

type psMigrationTestSuite struct {
	suite.Suite

	db     *rocksdb.RocksDB
	boltdb *bbolt.DB
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(psMigrationTestSuite))
}

func (suite *psMigrationTestSuite) SetupTest() {
	suite.db = rocksdbtest.RocksDBForT(suite.T())
	suite.boltdb = testutils.DBForT(suite.T())
}

func (suite *psMigrationTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.db)
	testutils.TearDownDB(suite.boltdb)
}

func (suite *psMigrationTestSuite) TestMigration() {

	var psToUpsert []*storage.PermissionSet
	psToUpsert = append(psToUpsert, unmigratedPSs...)
	psToUpsert = append(psToUpsert, alreadyMigratedPSs...)

	for _, initial := range psToUpsert {
		data, err := proto.Marshal(initial)
		suite.NoError(err)

		key := rocksdbmigration.GetPrefixedKey(psPrefix, []byte(initial.GetId()))
		suite.NoError(suite.db.Put(writeOpts, key, data))
	}

	suite.NoError(suite.boltdb.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucket(authPluginBucket)
		return err
	}))

	dbs := &types.Databases{
		BoltDB:  suite.boltdb,
		RocksDB: suite.db.DB,
	}

	suite.NoError(migration.Run(dbs))

	var allPSsAfterMigration []*storage.PermissionSet

	for _, existing := range psToUpsert {
		msg, exists, err := rockshelper.ReadFromRocksDB(suite.db.DB, readOpts, &storage.PermissionSet{}, psPrefix, []byte(existing.GetId()))
		suite.NoError(err)
		suite.True(exists)

		allPSsAfterMigration = append(allPSsAfterMigration, msg.(*storage.PermissionSet))
	}

	var expectedPSsAfterMigration []*storage.PermissionSet
	expectedPSsAfterMigration = append(expectedPSsAfterMigration, unmigratedPSsAfterMigration...)
	expectedPSsAfterMigration = append(expectedPSsAfterMigration, alreadyMigratedPSs...)

	suite.ElementsMatch(expectedPSsAfterMigration, allPSsAfterMigration)

	suite.NoError(suite.boltdb.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(authPluginBucket)
		suite.Nil(bucket)
		return nil
	}))
}

func (suite *psMigrationTestSuite) TestMigrationOnCleanDB() {
	dbs := &types.Databases{
		BoltDB:  suite.boltdb,
		RocksDB: suite.db.DB,
	}
	suite.NoError(migration.Run(dbs))
}
