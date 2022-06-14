package m93tom94

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(rolesUpdateTestSuite))
}

type rolesUpdateTestSuite struct {
	suite.Suite

	rocksDB *rocksdb.RocksDB
	db      *gorocksdb.DB
}

var _ suite.TearDownTestSuite = (*rolesUpdateTestSuite)(nil)

func (suite *rolesUpdateTestSuite) SetupTest() {
	var err error
	suite.rocksDB, err = rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.db = suite.rocksDB.DB
}

func (suite *rolesUpdateTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.rocksDB)
}

func (suite *rolesUpdateTestSuite) TestRolesUpdate() {
	roles := []*storage.Role{
		{Name: "Without scope"},
		{Name: "With scope", AccessScopeId: "some.scope"},
	}
	writeOpts := gorocksdb.NewDefaultWriteOptions()

	for _, role := range roles {
		value, err := proto.Marshal(role)
		suite.NoError(err)
		suite.NoError(suite.db.Put(writeOpts,
			rocksdbmigration.GetPrefixedKey(rolesBucket, []byte(role.Name)),
			value))
	}

	err := updateRoles(suite.db)
	suite.NoError(err)

	readOpts := gorocksdb.NewDefaultReadOptions()
	for _, oldRole := range roles {
		msg, exists, err := rockshelper.ReadFromRocksDB(suite.db, readOpts,
			&storage.Role{}, rolesBucket, []byte(oldRole.Name))
		suite.NoError(err)
		suite.True(exists)
		newRole := msg.(*storage.Role)
		if oldRole.AccessScopeId == "" {
			suite.Equal(unrestrictedScopeID, newRole.AccessScopeId)
		} else {
			suite.Equal(oldRole.AccessScopeId, newRole.AccessScopeId)
		}
	}
}
