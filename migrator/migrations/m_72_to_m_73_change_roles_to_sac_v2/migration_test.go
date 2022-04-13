package m72tom73

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/migrator/migrations/rocksdbmigration"
	dbTypes "github.com/stackrox/stackrox/migrator/types"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/stackrox/stackrox/pkg/testutils"
	"github.com/stackrox/stackrox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
	bolt "go.etcd.io/bbolt"
)

var (
	psID1          = "io.stackrox.authz.permissionset.94ac7bfe-f9b2-402e-b4f2-bfda480e1a15"
	oldFormatRoles = map[string]*storage.Role{
		"A": {
			Name: "A",
		},
		"B": {
			Name: "B",
			ResourceToAccess: map[string]storage.Access{
				"Cluster": storage.Access_READ_WRITE_ACCESS,
				"Image":   storage.Access_READ_ACCESS,
			},
		},
	}
	newFormatRoles = map[string]*storage.Role{
		"C": {
			Name:            "C",
			PermissionSetId: psID1,
		},
	}
	newFormatPermissionSets = map[string]*storage.PermissionSet{
		psID1: {
			Id:   psID1,
			Name: "for_role_C",
			ResourceToAccess: map[string]storage.Access{
				"Node":          storage.Access_READ_WRITE_ACCESS,
				"NetworkPolicy": storage.Access_READ_ACCESS,
			},
		},
	}
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(clusterRocksDBMigrationTestSuite))
}

type clusterRocksDBMigrationTestSuite struct {
	suite.Suite

	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (suite *clusterRocksDBMigrationTestSuite) SetupTest() {
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

func (suite *clusterRocksDBMigrationTestSuite) TearDownTest() {
	testutils.TearDownDB(suite.databases.BoltDB)
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *clusterRocksDBMigrationTestSuite) TestRolesGlobalAccessMigration() {
	rolesToUpsert := make(map[string]*storage.Role)
	for k, v := range oldFormatRoles {
		rolesToUpsert[k] = v
	}
	for k, v := range newFormatRoles {
		rolesToUpsert[k] = v
	}

	boltDB := suite.databases.BoltDB
	rocksDB := suite.databases.RocksDB

	// Insert permission sets for roles in the new format.
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()
	for _, ps := range newFormatPermissionSets {
		bytes, err := proto.Marshal(ps)
		suite.NoError(err)
		rocksWriteBatch.Put(rocksdbmigration.GetPrefixedKey(psBucket, []byte(ps.Id)), bytes)
	}
	suite.NoError(rocksDB.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch))

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

	// Extract migrated roles in the old format from database.
	oldFormatRolesAfterMigration := suite.extractRoles(boltDB, oldFormatRoles)
	// Extract roles in the new format from database.
	newFormatRolesAfterMigration := suite.extractRoles(boltDB, newFormatRoles)

	// Extract permission sets for roles in the old format from database.
	oldFormatPermissionSets := suite.extractPermissionSets(rocksDB, oldFormatRolesAfterMigration)
	// Extract permission sets for roles in the new format from database.
	newFormatPermissionSetsAfter := suite.extractPermissionSets(rocksDB, newFormatRolesAfterMigration)

	suite.Equal(len(oldFormatRoles), len(oldFormatRolesAfterMigration))
	suite.Equal(len(newFormatRoles), len(newFormatRolesAfterMigration))
	suite.Equal(len(oldFormatRoles), len(oldFormatPermissionSets))
	suite.Equal(len(newFormatRoles), len(newFormatPermissionSetsAfter))
	suite.Equal(len(newFormatPermissionSets), len(newFormatPermissionSetsAfter))

	suite.Equal(newFormatRoles, newFormatRolesAfterMigration)
	suite.Equal(newFormatPermissionSets, newFormatPermissionSetsAfter)
	for name, role := range oldFormatRoles {
		migratedRole, exists := oldFormatRolesAfterMigration[name]
		suite.True(exists)
		ps, exists := oldFormatPermissionSets[migratedRole.GetPermissionSetId()]
		suite.True(exists)
		suite.Equal(role.GetResourceToAccess(), ps.GetResourceToAccess())
	}
}

func (suite *clusterRocksDBMigrationTestSuite) extractRoles(boltDB *bolt.DB, roles map[string]*storage.Role) map[string]*storage.Role {
	rolesAfterMigration := make(map[string]*storage.Role)
	suite.NoError(boltDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(rolesBucket)
		if bucket == nil {
			return errors.New("bucket does not exist")
		}
		return bucket.ForEach(func(k, v []byte) error {
			if _, exists := roles[string(k)]; !exists {
				return nil
			}
			role := &storage.Role{}
			if err := proto.Unmarshal(v, role); err != nil {
				return err
			}
			if string(k) != role.GetName() {
				return errors.Errorf("Name mismatch: %s vs %s", k, role.GetName())
			}
			rolesAfterMigration[role.GetName()] = role
			return nil
		})
	}))
	return rolesAfterMigration
}

func (suite *clusterRocksDBMigrationTestSuite) extractPermissionSets(db *gorocksdb.DB, roles map[string]*storage.Role) map[string]*storage.PermissionSet {
	it := db.NewIterator(gorocksdb.NewDefaultReadOptions())
	defer it.Close()

	permissionSets := make(map[string]*storage.PermissionSet)

	for it.Seek(psBucket); it.ValidForPrefix(psBucket); it.Next() {
		id := rocksdbmigration.GetIDFromPrefixedKey(psBucket, it.Key().Copy())
		exists := false
		for _, role := range roles {
			exists = exists || role.GetPermissionSetId() == string(id)
		}
		if !exists {
			continue
		}
		var ps storage.PermissionSet
		if err := proto.Unmarshal(it.Value().Data(), &ps); err != nil {
			suite.NoError(err)
		}
		permissionSets[string(id)] = &ps
	}
	return permissionSets
}
