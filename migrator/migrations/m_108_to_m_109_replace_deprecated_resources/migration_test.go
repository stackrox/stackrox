package m108tom109

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

const (
	// Non-replaced resources
	Alert                            = "Alert"
	CVE                              = "CVE"
	Cluster                          = "Cluster"
	Deployment                       = "Deployment"
	Detection                        = "Detection"
	K8sRole                          = "K8sRole"
	K8sRoleBinding                   = "K8sRoleBinding"
	K8sSubject                       = "K8sSubject"
	Namespace                        = "Namespace"
	NetworkGraph                     = "NetworkGraph"
	NetworkPolicy                    = "NetworkPolicy"
	Node                             = "Node"
	Policy                           = "Policy"
	Secret                           = "Secret"
	ServiceAccount                   = "ServiceAccount"
	VulnerabilityManagementApprovals = "VulnerabilityManagementApprovals"
	VulnerabilityManagementRequests  = "VulnerabilityManagementRequests"
	VulnerabilityReports             = "VulnerabilityReports"
	WatchedImage                     = "WatchedImage"
	// Non-replaced internal resources
	ComplianceOperator = "ComplianceOperator"
	InstallationInfo   = "InstallationInfo"
	Version            = "Version"
)

var (
	UnmigratedPermissionSets = []*storage.PermissionSet{
		{
			Id:          "AA4618AC-EDD7-4756-828F-FA8424DE138E",
			Name:        "TestSet01",
			Description: "PermissionSet with no resource that requires replacement",
			ResourceToAccess: map[string]storage.Access{
				resources.Access.String(): storage.Access_READ_ACCESS,
				resources.Alert.String():  storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "6C618B1C-8919-4939-8A90-082EC9A90DA4",
			Name:        "TestSet02",
			Description: "PermissionSet with a replaced resource for which the replacement resource is not yet set",
			ResourceToAccess: map[string]storage.Access{
				resources.NetworkBaseline.String(): storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "97A38C2D-D11D-4355-AD80-732F3661EC4B",
			Name:        "TestSet03",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with lower access",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_NO_ACCESS,
				resources.NetworkBaseline.String():     storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "7035AD8F-E811-484B-AE36-E5877325B3F0",
			Name:        "TestSet04",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_READ_ACCESS,
				resources.NetworkBaseline.String():     storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "589ADE2F-BD33-4BA7-9821-3818832C5A79",
			Name:        "TestSet05",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_READ_WRITE_ACCESS,
				resources.NetworkBaseline.String():     storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "A78B24E1-F0ED-456A-B679-BADF4C47F654",
			Name:        "TestSet06",
			Description: "PermissionSet with two replaced resources for which the replacement resource is not yet set",
			ResourceToAccess: map[string]storage.Access{
				resources.NetworkBaseline.String():  storage.Access_READ_WRITE_ACCESS,
				resources.ProcessWhitelist.String(): storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "C0EE37B8-36F5-4070-AD8D-34A44A1D4ABB",
			Name:        "TestSet07",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with higher access",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_READ_WRITE_ACCESS,
				resources.NetworkBaseline.String():     storage.Access_READ_ACCESS,
				resources.ProcessWhitelist.String():    storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "96D8F800-DACF-4FF2-8674-9AAD8230CF49",
			Name:        "TestSet08",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than one of the replaced resources",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_READ_ACCESS,
				resources.NetworkBaseline.String():     storage.Access_READ_ACCESS,
				resources.ProcessWhitelist.String():    storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "DBFF2131-811E-4F22-9386-449AF02B9053",
			Name:        "TestSet09",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_NO_ACCESS,
				resources.NetworkBaseline.String():     storage.Access_READ_ACCESS,
				resources.ProcessWhitelist.String():    storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "98D88DBA-1996-40BA-BC4D-953E3D60E35A",
			Name:        "TestSet10",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_NO_ACCESS,
				resources.NetworkBaseline.String():     storage.Access_READ_WRITE_ACCESS,
				resources.ProcessWhitelist.String():    storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "E0F50165-9914-4D0E-8C37-E8C8D482C904",
			Name:        "TestSet11",
			Description: "PermissionSet with access defined for all existing resource types",
			ResourceToAccess: map[string]storage.Access{
				// Replacing resources
				resources.Access.String():              storage.Access_READ_ACCESS,
				resources.Administration.String():      storage.Access_READ_ACCESS,
				resources.Compliance.String():          storage.Access_NO_ACCESS,
				resources.DeploymentExtension.String(): storage.Access_NO_ACCESS,
				resources.Image.String():               storage.Access_NO_ACCESS,
				resources.Integration.String():         storage.Access_READ_WRITE_ACCESS,
				// Replaced resources
				resources.AllComments.String():           storage.Access_NO_ACCESS,
				resources.APIToken.String():              storage.Access_READ_WRITE_ACCESS,
				resources.AuthProvider.String():          storage.Access_NO_ACCESS,
				resources.BackupPlugins.String():         storage.Access_NO_ACCESS,
				resources.ComplianceRuns.String():        storage.Access_READ_WRITE_ACCESS,
				resources.ComplianceRunSchedule.String(): storage.Access_NO_ACCESS,
				resources.Config.String():                storage.Access_READ_ACCESS,
				resources.DebugLogs.String():             storage.Access_READ_WRITE_ACCESS,
				resources.Group.String():                 storage.Access_NO_ACCESS,
				resources.ImageComponent.String():        storage.Access_READ_ACCESS,
				resources.ImageIntegration.String():      storage.Access_NO_ACCESS,
				resources.Indicator.String():             storage.Access_READ_ACCESS,
				resources.Licenses.String():              storage.Access_READ_ACCESS,
				resources.NetworkBaseline.String():       storage.Access_READ_ACCESS,
				resources.NetworkGraphConfig.String():    storage.Access_NO_ACCESS,
				resources.Notifier.String():              storage.Access_READ_WRITE_ACCESS,
				resources.ProbeUpload.String():           storage.Access_READ_WRITE_ACCESS,
				resources.ProcessWhitelist.String():      storage.Access_READ_WRITE_ACCESS,
				resources.Risk.String():                  storage.Access_READ_WRITE_ACCESS,
				resources.Role.String():                  storage.Access_READ_WRITE_ACCESS,
				resources.ScannerBundle.String():         storage.Access_NO_ACCESS,
				resources.ScannerDefinitions.String():    storage.Access_READ_ACCESS,
				resources.SensorUpgradeConfig.String():   storage.Access_READ_ACCESS,
				resources.ServiceIdentity.String():       storage.Access_READ_ACCESS,
				resources.SignatureIntegration.String():  storage.Access_NO_ACCESS,
				resources.User.String():                  storage.Access_READ_WRITE_ACCESS,
				// Non-replaced resources
				resources.Alert.String():                            storage.Access_NO_ACCESS,
				resources.CVE.String():                              storage.Access_NO_ACCESS,
				resources.Cluster.String():                          storage.Access_READ_WRITE_ACCESS,
				resources.Deployment.String():                       storage.Access_READ_ACCESS,
				resources.Detection.String():                        storage.Access_NO_ACCESS,
				resources.K8sRole.String():                          storage.Access_NO_ACCESS,
				resources.K8sRoleBinding.String():                   storage.Access_READ_ACCESS,
				resources.K8sSubject.String():                       storage.Access_NO_ACCESS,
				resources.Namespace.String():                        storage.Access_READ_WRITE_ACCESS,
				resources.NetworkGraph.String():                     storage.Access_NO_ACCESS,
				resources.NetworkPolicy.String():                    storage.Access_NO_ACCESS,
				resources.Node.String():                             storage.Access_READ_ACCESS,
				resources.Policy.String():                           storage.Access_READ_WRITE_ACCESS,
				resources.Secret.String():                           storage.Access_READ_ACCESS,
				resources.ServiceAccount.String():                   storage.Access_NO_ACCESS,
				resources.VulnerabilityManagementApprovals.String(): storage.Access_READ_WRITE_ACCESS,
				resources.VulnerabilityManagementRequests.String():  storage.Access_READ_WRITE_ACCESS,
				resources.VulnerabilityReports.String():             storage.Access_READ_ACCESS,
				resources.WatchedImage.String():                     storage.Access_READ_ACCESS,
				// Internal resources
				resources.ComplianceOperator.String(): storage.Access_READ_ACCESS,
				resources.InstallationInfo.String():   storage.Access_READ_ACCESS,
				resources.Version.String():            storage.Access_READ_WRITE_ACCESS,
			},
		},
	}

	MigratedPermissionSets = []*storage.PermissionSet{
		{
			Id:          "AA4618AC-EDD7-4756-828F-FA8424DE138E",
			Name:        "TestSet01",
			Description: "PermissionSet with no resource that requires replacement",
			ResourceToAccess: map[string]storage.Access{
				resources.Access.String(): storage.Access_READ_ACCESS,
				resources.Alert.String():  storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "6C618B1C-8919-4939-8A90-082EC9A90DA4",
			Name:        "TestSet02",
			Description: "PermissionSet with a replaced resource for which the replacement resource is not yet set",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "97A38C2D-D11D-4355-AD80-732F3661EC4B",
			Name:        "TestSet03",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with lower access",
			ResourceToAccess: map[string]storage.Access{
				// Keeps the access of the replaced resource which is higher
				resources.DeploymentExtension.String(): storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "7035AD8F-E811-484B-AE36-E5877325B3F0",
			Name:        "TestSet04",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "589ADE2F-BD33-4BA7-9821-3818832C5A79",
			Name:        "TestSet05",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				// Keep the access of the replacing resource which is higher
				resources.DeploymentExtension.String(): storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "A78B24E1-F0ED-456A-B679-BADF4C47F654",
			Name:        "TestSet06",
			Description: "PermissionSet with two replaced resources for which the replacement resource is not yet set",
			ResourceToAccess: map[string]storage.Access{
				// Keep the highest access of the replaced resources
				resources.DeploymentExtension.String(): storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "C0EE37B8-36F5-4070-AD8D-34A44A1D4ABB",
			Name:        "TestSet07",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with higher access",
			ResourceToAccess: map[string]storage.Access{
				// Keep the access of the replacing resource which is higher
				resources.DeploymentExtension.String(): storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "96D8F800-DACF-4FF2-8674-9AAD8230CF49",
			Name:        "TestSet08",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than one of the replaced resources",
			ResourceToAccess: map[string]storage.Access{
				// Keep the highest access of the replaced resources
				resources.DeploymentExtension.String(): storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "DBFF2131-811E-4F22-9386-449AF02B9053",
			Name:        "TestSet09",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "98D88DBA-1996-40BA-BC4D-953E3D60E35A",
			Name:        "TestSet10",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				resources.DeploymentExtension.String(): storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "E0F50165-9914-4D0E-8C37-E8C8D482C904",
			Name:        "TestSet11",
			Description: "PermissionSet with access defined for all existing resource types",
			ResourceToAccess: map[string]storage.Access{
				// Replacing resources
				resources.Access.String():              storage.Access_READ_WRITE_ACCESS,
				resources.Administration.String():      storage.Access_READ_WRITE_ACCESS,
				resources.Compliance.String():          storage.Access_READ_WRITE_ACCESS,
				resources.DeploymentExtension.String(): storage.Access_READ_WRITE_ACCESS,
				resources.Image.String():               storage.Access_READ_ACCESS,
				resources.Integration.String():         storage.Access_READ_WRITE_ACCESS,
				// Non-replaced resources
				resources.Alert.String():                            storage.Access_NO_ACCESS,
				resources.CVE.String():                              storage.Access_NO_ACCESS,
				resources.Cluster.String():                          storage.Access_READ_WRITE_ACCESS,
				resources.Deployment.String():                       storage.Access_READ_ACCESS,
				resources.Detection.String():                        storage.Access_NO_ACCESS,
				resources.K8sRole.String():                          storage.Access_NO_ACCESS,
				resources.K8sRoleBinding.String():                   storage.Access_READ_ACCESS,
				resources.K8sSubject.String():                       storage.Access_NO_ACCESS,
				resources.Namespace.String():                        storage.Access_READ_WRITE_ACCESS,
				resources.NetworkGraph.String():                     storage.Access_NO_ACCESS,
				resources.NetworkPolicy.String():                    storage.Access_NO_ACCESS,
				resources.Node.String():                             storage.Access_READ_ACCESS,
				resources.Policy.String():                           storage.Access_READ_WRITE_ACCESS,
				resources.Secret.String():                           storage.Access_READ_ACCESS,
				resources.ServiceAccount.String():                   storage.Access_NO_ACCESS,
				resources.VulnerabilityManagementApprovals.String(): storage.Access_READ_WRITE_ACCESS,
				resources.VulnerabilityManagementRequests.String():  storage.Access_READ_WRITE_ACCESS,
				resources.VulnerabilityReports.String():             storage.Access_READ_ACCESS,
				resources.WatchedImage.String():                     storage.Access_READ_ACCESS,
				// Internal resources
				resources.ComplianceOperator.String(): storage.Access_READ_ACCESS,
				resources.InstallationInfo.String():   storage.Access_READ_ACCESS,
				resources.Version.String():            storage.Access_READ_WRITE_ACCESS,
			},
		},
	}
)

type psMigrationTestSuite struct {
	suite.Suite

	db *rocksdb.RocksDB
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(psMigrationTestSuite))
}

func (suite *psMigrationTestSuite) SetupTest() {
	suite.db = rocksdbtest.RocksDBForT(suite.T())
}

func (suite *psMigrationTestSuite) TearDownTest() {
	rocksdbtest.TearDownRocksDB(suite.db)
}

func (suite *psMigrationTestSuite) TestMigration() {

	for _, initial := range UnmigratedPermissionSets {
		data, err := proto.Marshal(initial)
		suite.NoError(err)

		key := rocksdbmigration.GetPrefixedKey(prefix, []byte(initial.GetId()))
		suite.NoError(suite.db.Put(writeOpts, key, data))
	}

	dbs := &types.Databases{
		RocksDB: suite.db.DB,
	}

	suite.NoError(migration.Run(dbs))

	var allPSsAfterMigration []*storage.PermissionSet

	for _, existing := range UnmigratedPermissionSets {
		msg, exists, err := rockshelper.ReadFromRocksDB(suite.db.DB, readOpts, &storage.PermissionSet{}, prefix, []byte(existing.GetId()))
		suite.NoError(err)
		suite.True(exists)

		allPSsAfterMigration = append(allPSsAfterMigration, msg.(*storage.PermissionSet))
	}

	var expectedPSsAfterMigration []*storage.PermissionSet
	expectedPSsAfterMigration = append(expectedPSsAfterMigration, MigratedPermissionSets...)

	suite.ElementsMatch(expectedPSsAfterMigration, allPSsAfterMigration)
}

func (suite *psMigrationTestSuite) TestMigrationOnCleanDB() {
	dbs := &types.Databases{
		RocksDB: suite.db.DB,
	}
	suite.NoError(migration.Run(dbs))
}
