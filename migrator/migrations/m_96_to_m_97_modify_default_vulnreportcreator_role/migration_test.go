package m96tom97

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
	"github.com/tecbot/gorocksdb"
)

func TestMigration(t *testing.T) {
	suite.Run(t, new(vulnReporterRoleUpdateTestSuite))
}

type vulnReporterRoleUpdateTestSuite struct {
	suite.Suite

	rocksDB *rocksdb.RocksDB
	db      *gorocksdb.DB
}

var _ suite.TearDownTestSuite = (*vulnReporterRoleUpdateTestSuite)(nil)

func (suite *vulnReporterRoleUpdateTestSuite) SetupTest() {
	var err error
	suite.rocksDB, err = rocksdb.NewTemp(suite.T().Name())
	suite.NoError(err)

	suite.db = suite.rocksDB.DB
}

func (suite *vulnReporterRoleUpdateTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.rocksDB)
}

func (suite *vulnReporterRoleUpdateTestSuite) TestRolesUpdateForVulnReporterRole() {
	oldPermissions := map[string]storage.Access{
		"Image":                storage.Access_READ_ACCESS,
		"Notifier":             storage.Access_READ_WRITE_ACCESS,
		"Role":                 storage.Access_READ_ACCESS,
		"VulnerabilityReports": storage.Access_READ_WRITE_ACCESS,
	}
	expectedNewPermissions := map[string]storage.Access{
		"Image":                storage.Access_READ_ACCESS,
		"Notifier":             storage.Access_READ_ACCESS,
		"Role":                 storage.Access_READ_ACCESS,
		"VulnerabilityReports": storage.Access_READ_WRITE_ACCESS,
	}

	permissionSet := &storage.PermissionSet{
		Id:               "vulnreporter",
		Name:             vulnReporterRoleName,
		Description:      "For users: use it to create and manage vulnerability reporting configurations for scheduled vulnerability reports",
		ResourceToAccess: oldPermissions,
	}

	vulnReporterRole := &storage.Role{
		Name:            vulnReporterRoleName,
		Description:     permissionSet.Description,
		AccessScopeId:   "some-id",
		PermissionSetId: permissionSet.Id,
	}

	randomPermissionSet := &storage.PermissionSet{
		Id:               "random-id",
		Name:             vulnReporterRoleName,
		Description:      "For users: use it to create and manage vulnerability reporting configurations for scheduled vulnerability reports",
		ResourceToAccess: oldPermissions,
	}

	randomRole := &storage.Role{
		Name:            "random-role",
		Description:     randomPermissionSet.Description,
		AccessScopeId:   "some-id",
		PermissionSetId: randomPermissionSet.Id,
	}

	writeOpts := gorocksdb.NewDefaultWriteOptions()
	value, err := proto.Marshal(randomPermissionSet)
	suite.NoError(err)
	suite.NoError(suite.db.Put(writeOpts,
		rocksdbmigration.GetPrefixedKey(permissionsBucket, []byte(randomPermissionSet.Id)),
		value))

	value, err = proto.Marshal(randomRole)
	suite.NoError(err)
	suite.NoError(suite.db.Put(writeOpts,
		rocksdbmigration.GetPrefixedKey(rolesBucket, []byte(randomRole.Name)),
		value))

	value, err = proto.Marshal(permissionSet)
	suite.NoError(err)
	suite.NoError(suite.db.Put(writeOpts,
		rocksdbmigration.GetPrefixedKey(permissionsBucket, []byte(permissionSet.Id)),
		value))

	value, err = proto.Marshal(vulnReporterRole)
	suite.NoError(err)
	suite.NoError(suite.db.Put(writeOpts,
		rocksdbmigration.GetPrefixedKey(rolesBucket, []byte(vulnReporterRole.Name)),
		value))

	err = updateDefaultPermissionsForVulnCreatorRole(suite.db)
	suite.NoError(err)

	readOpts := gorocksdb.NewDefaultReadOptions()
	msg, exists, err := rockshelper.ReadFromRocksDB(suite.db, readOpts,
		&storage.PermissionSet{}, permissionsBucket, []byte(permissionSet.Id))
	suite.NoError(err)
	suite.True(exists)
	newRolePermissions := msg.(*storage.PermissionSet)
	suite.Assert().Equal(4, len(newRolePermissions.ResourceToAccess))
	suite.Assert().True(reflect.DeepEqual(newRolePermissions.ResourceToAccess, expectedNewPermissions))

	// random role
	msg, exists, err = rockshelper.ReadFromRocksDB(suite.db, readOpts,
		&storage.PermissionSet{}, permissionsBucket, []byte(randomPermissionSet.Id))
	suite.NoError(err)
	suite.True(exists)
	newRolePermissions = msg.(*storage.PermissionSet)
	suite.Assert().Equal(4, len(newRolePermissions.ResourceToAccess))
	suite.Assert().True(reflect.DeepEqual(newRolePermissions.ResourceToAccess, oldPermissions))
}
