package m76to77

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
)

var (
	psID1         = "io.stackrox.authz.permissionset.94ac7bfe-f9b2-402e-b4f2-bfda480e1a15"
	asID1         = "io.stackrox.authz.accessscope.94ac7bfe-f9b2-402e-b4f2-bfda480e1a15"
	rolesToUpsert = map[string]*storage.Role{
		"C": {
			Name:            "C",
			Description:     "This is description",
			PermissionSetId: psID1,
			AccessScopeId:   asID1,
		},
		"B": {
			Name:        "B",
			Description: "This is not a description",
		},
	}
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(rolesRocksDBMigrationTestSuite))
}

type rolesRocksDBMigrationTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (suite *rolesRocksDBMigrationTestSuite) SetupTest() {
	boltdb := testutils.DBForT(suite.T())
	suite.NoError(boltdb.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(rolesBucket); err != nil {
			return err
		}
		return nil
	}))

	rocksDB, err := rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.db = rocksDB
	suite.databases = &dbTypes.Databases{BoltDB: boltdb, RocksDB: rocksDB.DB}
}

func (suite *rolesRocksDBMigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.databases.BoltDB)
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *rolesRocksDBMigrationTestSuite) TestRolesMigrationToRocksDB() {
	boltDB := suite.databases.BoltDB
	rocksDB := suite.databases.RocksDB

	// Insert all roles.
	suite.NoError(boltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(rolesBucket)
		for _, role := range rolesToUpsert {
			bytes, err := proto.Marshal(role)
			if err != nil {
				return err
			}
			if err := bucket.Put([]byte(role.GetName()), bytes); err != nil {
				return err
			}
		}
		return nil
	}))

	// Run migration.
	suite.NoError(migrateRoles(boltDB, rocksDB))

	rolesAfterMigration := make(map[string]*storage.Role)

	it := rocksDB.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()
	for it.Seek(rolesBucket); it.ValidForPrefix(rolesBucket); it.Next() {
		var role storage.Role
		if err := proto.Unmarshal(it.Value().Data(), &role); err != nil {
			suite.NoError(err)
		}
		rolesAfterMigration[role.GetName()] = &role
	}

	suite.Equal(rolesToUpsert, rolesAfterMigration)

	// Verify roles bucket is deleted in boltdb.
	suite.NoError(boltDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(rolesBucket)
		if bucket != nil {
			return errors.New("roles bucket is not deleted when it should")
		}
		return nil
	}))
}
