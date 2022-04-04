package m96tom97

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
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

func (suite *vulnReporterRoleUpdateTestSuite) TestRolesUpdate() {
	oldPermissions := []permissions.ResourceWithAccess{
		permissions.View(resources.VulnerabilityReports),   // required for vuln report configurations
		permissions.Modify(resources.VulnerabilityReports), // required for vuln report configurations
		permissions.View(resources.Role),                   // required for scopes
		permissions.View(resources.Image),                  // required to gather CVE data for the report
		permissions.View(resources.Notifier),               // required for vuln report configurations
		permissions.Modify(resources.Notifier),             // required for vuln report configurations
	}

	expectedNewPermissions := []permissions.ResourceWithAccess{
		permissions.View(resources.VulnerabilityReports),   // required for vuln report configurations
		permissions.Modify(resources.VulnerabilityReports), // required for vuln report configurations
		permissions.View(resources.Role),                   // required for scopes
		permissions.View(resources.Image),                  // required to gather CVE data for the report
		permissions.View(resources.Notifier),               // required for vuln report configurations
	}

	permissionSet := &storage.PermissionSet{
		Id:          rolePkg.EnsureValidPermissionSetID("vulnreporter"),
		Name:        rolePkg.VulnReporter,
		Description: "For users: use it to create and manage vulnerability reporting configurations for scheduled vulnerability reports",

		ResourceToAccess: permissionsUtils.FromResourcesWithAccess(oldPermissions...),
	}
	role := &storage.Role{
		Name:            rolePkg.VulnReporter,
		Description:     permissionSet.Description,
		AccessScopeId:   rolePkg.AccessScopeIncludeAll.GetId(),
		PermissionSetId: permissionSet.Id,
	}

	writeOpts := gorocksdb.NewDefaultWriteOptions()
	value, err := proto.Marshal(permissionSet)
	suite.NoError(err)
	suite.NoError(suite.db.Put(writeOpts,
		rocksdbmigration.GetPrefixedKey(permissionsBucket, []byte(permissionSet.Id)),
		value))

	value, err = proto.Marshal(role)
	suite.NoError(err)
	suite.NoError(suite.db.Put(writeOpts,
		rocksdbmigration.GetPrefixedKey(rolesBucket, []byte(role.Name)),
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
	suite.Assert().True(reflect.DeepEqual(newRolePermissions.ResourceToAccess, permissionsUtils.FromResourcesWithAccess(expectedNewPermissions...)))
}
