package m90tom91

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	dbTypes "github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(vulnMgmtPermisssionTestSuite))
}

type vulnMgmtPermisssionTestSuite struct {
	suite.Suite
	db        *rocksdb.RocksDB
	databases *dbTypes.Databases
}

func (suite *vulnMgmtPermisssionTestSuite) SetupTest() {
	rocksDB, err := rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.db = rocksDB
	suite.databases = &dbTypes.Databases{RocksDB: rocksDB.DB}
}

func (suite *vulnMgmtPermisssionTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *vulnMgmtPermisssionTestSuite) TestMigrationToVulnMgmtPermission() {
	permissionSets := []*storage.PermissionSet{
		{
			Id: "set1",
			ResourceToAccess: map[string]storage.Access{
				imageResource: storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id: "set2",
			ResourceToAccess: map[string]storage.Access{
				imageResource: storage.Access_READ_ACCESS,
			},
		},
		{
			Id: "set3",
			ResourceToAccess: map[string]storage.Access{
				"blah": storage.Access_READ_WRITE_ACCESS,
			},
		},
		{},
	}

	expectedPermissionSets := []*storage.PermissionSet{
		{
			Id: "set1",
			ResourceToAccess: map[string]storage.Access{
				imageResource:             storage.Access_READ_WRITE_ACCESS,
				vulnMgmtRequestsResource:  storage.Access_READ_WRITE_ACCESS,
				vulnMgmtApprovalsResource: storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id: "set2",
			ResourceToAccess: map[string]storage.Access{
				imageResource: storage.Access_READ_ACCESS,
			},
		},
		{
			Id: "set3",
			ResourceToAccess: map[string]storage.Access{
				"blah": storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	for _, obj := range permissionSets {
		key := rocksdbmigration.GetPrefixedKey(permissionSetPrefix, []byte(obj.GetId()))
		value, err := proto.Marshal(obj)
		suite.NoError(err)
		suite.NoError(suite.databases.RocksDB.Put(writeOpts, key, value))
	}

	err := updateVulnSnoozePermissions(suite.databases)
	suite.NoError(err)

	for _, p := range expectedPermissionSets {
		msg, exists, err := rockshelper.ReadFromRocksDB(suite.databases.RocksDB, readOpts, &storage.PermissionSet{}, permissionSetPrefix, []byte(p.GetId()))
		suite.NoError(err)
		suite.True(exists)
		suite.EqualValues(p, msg.(*storage.PermissionSet))
	}
}
