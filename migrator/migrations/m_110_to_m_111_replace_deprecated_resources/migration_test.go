package m110tom111

import (
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/rockshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/testutils/rocksdbtest"
	"github.com/stretchr/testify/suite"
)

// Non-replaced resources
const (
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
	Role                             = "Role"
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

// Future-replacement resources
const (
	Administration = "Administration"
	Compliance     = "Compliance"
)

// To-be-replaced-later resources
const (
	AllComments           = "AllComments"
	ComplianceRuns        = "ComplianceRuns"
	ComplianceRunSchedule = "ComplianceRunSchedule"
	Config                = "Config"
	DebugLogs             = "DebugLogs"
	NetworkGraphConfig    = "NetworkGraphConfig"
	ProbeUpload           = "ProbeUpload"
	ScannerBundle         = "ScannerBundle"
	ScannerDefinitions    = "ScannerDefinitions"
	SensorUpgradeConfig   = "SensorUpgradeConfig"
	ServiceIdentity       = "ServiceIdentity"
)

var (
	UnmigratedPermissionSets = []*storage.PermissionSet{
		{
			Id:          "AA4618AC-EDD7-4756-828F-FA8424DE138E",
			Name:        "TestSet01",
			Description: "PermissionSet with no resource that requires replacement",
			ResourceToAccess: map[string]storage.Access{
				Access: storage.Access_READ_ACCESS,
				Alert:  storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "6C618B1C-8919-4939-8A90-082EC9A90DA4",
			Name:        "TestSet02",
			Description: "PermissionSet with a replaced resource for which the replacement resource is not yet set",
			ResourceToAccess: map[string]storage.Access{
				NetworkBaseline: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "97A38C2D-D11D-4355-AD80-732F3661EC4B",
			Name:        "TestSet03",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with lower access",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_NO_ACCESS,
				NetworkBaseline:     storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "7035AD8F-E811-484B-AE36-E5877325B3F0",
			Name:        "TestSet04",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_READ_ACCESS,
				NetworkBaseline:     storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "589ADE2F-BD33-4BA7-9821-3818832C5A79",
			Name:        "TestSet05",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_READ_WRITE_ACCESS,
				NetworkBaseline:     storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "A78B24E1-F0ED-456A-B679-BADF4C47F654",
			Name:        "TestSet06",
			Description: "PermissionSet with two replaced resources for which the replacement resource is not yet set",
			ResourceToAccess: map[string]storage.Access{
				NetworkBaseline:  storage.Access_READ_WRITE_ACCESS,
				ProcessWhitelist: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "C0EE37B8-36F5-4070-AD8D-34A44A1D4ABB",
			Name:        "TestSet07",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with higher access",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_READ_WRITE_ACCESS,
				NetworkBaseline:     storage.Access_READ_ACCESS,
				ProcessWhitelist:    storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "96D8F800-DACF-4FF2-8674-9AAD8230CF49",
			Name:        "TestSet08",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than one of the replaced resources",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_READ_ACCESS,
				NetworkBaseline:     storage.Access_READ_ACCESS,
				ProcessWhitelist:    storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "DBFF2131-811E-4F22-9386-449AF02B9053",
			Name:        "TestSet09",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_NO_ACCESS,
				NetworkBaseline:     storage.Access_READ_ACCESS,
				ProcessWhitelist:    storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "98D88DBA-1996-40BA-BC4D-953E3D60E35A",
			Name:        "TestSet10",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_NO_ACCESS,
				NetworkBaseline:     storage.Access_READ_WRITE_ACCESS,
				ProcessWhitelist:    storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "E0F50165-9914-4D0E-8C37-E8C8D482C904",
			Name:        "TestSet11",
			Description: "PermissionSet with access defined for all existing resource types",
			ResourceToAccess: map[string]storage.Access{
				// Replacing resources
				Access:              storage.Access_READ_ACCESS,
				Administration:      storage.Access_READ_ACCESS,
				Compliance:          storage.Access_NO_ACCESS,
				DeploymentExtension: storage.Access_NO_ACCESS,
				Image:               storage.Access_NO_ACCESS,
				Integration:         storage.Access_READ_WRITE_ACCESS,
				// Replaced resources
				APIToken:             storage.Access_READ_WRITE_ACCESS,
				AuthProvider:         storage.Access_NO_ACCESS,
				BackupPlugins:        storage.Access_NO_ACCESS,
				Group:                storage.Access_NO_ACCESS,
				ImageComponent:       storage.Access_READ_ACCESS,
				ImageIntegration:     storage.Access_NO_ACCESS,
				Indicator:            storage.Access_READ_ACCESS,
				Licenses:             storage.Access_READ_ACCESS,
				NetworkBaseline:      storage.Access_READ_ACCESS,
				Notifier:             storage.Access_READ_WRITE_ACCESS,
				ProcessWhitelist:     storage.Access_READ_WRITE_ACCESS,
				Risk:                 storage.Access_READ_WRITE_ACCESS,
				SignatureIntegration: storage.Access_NO_ACCESS,
				User:                 storage.Access_READ_WRITE_ACCESS,
				// To-be-replaced-later resources
				AllComments:           storage.Access_NO_ACCESS,
				ComplianceRuns:        storage.Access_READ_WRITE_ACCESS,
				ComplianceRunSchedule: storage.Access_NO_ACCESS,
				Config:                storage.Access_READ_ACCESS,
				DebugLogs:             storage.Access_READ_WRITE_ACCESS,
				NetworkGraphConfig:    storage.Access_NO_ACCESS,
				ProbeUpload:           storage.Access_READ_WRITE_ACCESS,
				ScannerBundle:         storage.Access_NO_ACCESS,
				ScannerDefinitions:    storage.Access_READ_ACCESS,
				SensorUpgradeConfig:   storage.Access_READ_ACCESS,
				ServiceIdentity:       storage.Access_READ_ACCESS,
				// Non-replaced resources
				Alert:                            storage.Access_NO_ACCESS,
				CVE:                              storage.Access_NO_ACCESS,
				Cluster:                          storage.Access_READ_WRITE_ACCESS,
				Deployment:                       storage.Access_READ_ACCESS,
				Detection:                        storage.Access_NO_ACCESS,
				K8sRole:                          storage.Access_NO_ACCESS,
				K8sRoleBinding:                   storage.Access_READ_ACCESS,
				K8sSubject:                       storage.Access_NO_ACCESS,
				Namespace:                        storage.Access_READ_WRITE_ACCESS,
				NetworkGraph:                     storage.Access_NO_ACCESS,
				NetworkPolicy:                    storage.Access_NO_ACCESS,
				Node:                             storage.Access_READ_ACCESS,
				Policy:                           storage.Access_READ_WRITE_ACCESS,
				Role:                             storage.Access_READ_WRITE_ACCESS,
				Secret:                           storage.Access_READ_ACCESS,
				ServiceAccount:                   storage.Access_NO_ACCESS,
				VulnerabilityManagementApprovals: storage.Access_READ_WRITE_ACCESS,
				VulnerabilityManagementRequests:  storage.Access_READ_WRITE_ACCESS,
				VulnerabilityReports:             storage.Access_READ_ACCESS,
				WatchedImage:                     storage.Access_READ_ACCESS,
				// Internal resources
				ComplianceOperator: storage.Access_READ_ACCESS,
				InstallationInfo:   storage.Access_READ_ACCESS,
				Version:            storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "E79F2114-F949-411B-9F6D-4D38C1404642",
			Name:        "TestSet12",
			Description: "PermissionSet with access defined to read for all existing resource types except DebugLogs (Analyst)",
			ResourceToAccess: map[string]storage.Access{
				// Replacing resources
				Access:              storage.Access_READ_ACCESS,
				Administration:      storage.Access_READ_ACCESS,
				Compliance:          storage.Access_READ_ACCESS,
				DeploymentExtension: storage.Access_READ_ACCESS,
				Image:               storage.Access_READ_ACCESS,
				Integration:         storage.Access_READ_ACCESS,
				// Replaced resources
				APIToken:             storage.Access_READ_ACCESS,
				AuthProvider:         storage.Access_READ_ACCESS,
				BackupPlugins:        storage.Access_READ_ACCESS,
				Group:                storage.Access_READ_ACCESS,
				ImageComponent:       storage.Access_READ_ACCESS,
				ImageIntegration:     storage.Access_READ_ACCESS,
				Indicator:            storage.Access_READ_ACCESS,
				Licenses:             storage.Access_READ_ACCESS,
				NetworkBaseline:      storage.Access_READ_ACCESS,
				Notifier:             storage.Access_READ_ACCESS,
				ProcessWhitelist:     storage.Access_READ_ACCESS,
				Risk:                 storage.Access_READ_ACCESS,
				SignatureIntegration: storage.Access_READ_ACCESS,
				User:                 storage.Access_READ_ACCESS,
				// To-be-replaced-later resources
				AllComments:           storage.Access_READ_ACCESS,
				ComplianceRuns:        storage.Access_READ_ACCESS,
				ComplianceRunSchedule: storage.Access_READ_ACCESS,
				Config:                storage.Access_READ_ACCESS,
				DebugLogs:             storage.Access_NO_ACCESS,
				NetworkGraphConfig:    storage.Access_READ_ACCESS,
				ProbeUpload:           storage.Access_READ_ACCESS,
				ScannerBundle:         storage.Access_READ_ACCESS,
				ScannerDefinitions:    storage.Access_READ_ACCESS,
				SensorUpgradeConfig:   storage.Access_READ_ACCESS,
				ServiceIdentity:       storage.Access_READ_ACCESS,
				// Non-replaced resources
				Alert:                            storage.Access_READ_ACCESS,
				CVE:                              storage.Access_READ_ACCESS,
				Cluster:                          storage.Access_READ_ACCESS,
				Deployment:                       storage.Access_READ_ACCESS,
				Detection:                        storage.Access_READ_ACCESS,
				K8sRole:                          storage.Access_READ_ACCESS,
				K8sRoleBinding:                   storage.Access_READ_ACCESS,
				K8sSubject:                       storage.Access_READ_ACCESS,
				Namespace:                        storage.Access_READ_ACCESS,
				NetworkGraph:                     storage.Access_READ_ACCESS,
				NetworkPolicy:                    storage.Access_READ_ACCESS,
				Node:                             storage.Access_READ_ACCESS,
				Policy:                           storage.Access_READ_ACCESS,
				Role:                             storage.Access_READ_ACCESS,
				Secret:                           storage.Access_READ_ACCESS,
				ServiceAccount:                   storage.Access_READ_ACCESS,
				VulnerabilityManagementApprovals: storage.Access_READ_ACCESS,
				VulnerabilityManagementRequests:  storage.Access_READ_ACCESS,
				VulnerabilityReports:             storage.Access_READ_ACCESS,
				WatchedImage:                     storage.Access_READ_ACCESS,
				// Internal resources
				ComplianceOperator: storage.Access_READ_ACCESS,
				InstallationInfo:   storage.Access_READ_ACCESS,
				Version:            storage.Access_READ_ACCESS,
			},
		},
	}

	MigratedPermissionSets = []*storage.PermissionSet{
		{
			Id:          "AA4618AC-EDD7-4756-828F-FA8424DE138E",
			Name:        "TestSet01",
			Description: "PermissionSet with no resource that requires replacement",
			ResourceToAccess: map[string]storage.Access{
				Access: storage.Access_READ_ACCESS,
				Alert:  storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "6C618B1C-8919-4939-8A90-082EC9A90DA4",
			Name:        "TestSet02",
			Description: "PermissionSet with a replaced resource for which the replacement resource is not yet set",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "97A38C2D-D11D-4355-AD80-732F3661EC4B",
			Name:        "TestSet03",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with lower access",
			ResourceToAccess: map[string]storage.Access{
				// Keeps the access of the replacing resource which is lower
				DeploymentExtension: storage.Access_NO_ACCESS,
			},
		},
		{
			Id:          "7035AD8F-E811-484B-AE36-E5877325B3F0",
			Name:        "TestSet04",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "589ADE2F-BD33-4BA7-9821-3818832C5A79",
			Name:        "TestSet05",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				// Keep the access of the replaced resource which is lower
				DeploymentExtension: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "A78B24E1-F0ED-456A-B679-BADF4C47F654",
			Name:        "TestSet06",
			Description: "PermissionSet with two replaced resources for which the replacement resource is not yet set",
			ResourceToAccess: map[string]storage.Access{
				// Keep the lowest access of the replaced resources
				DeploymentExtension: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "C0EE37B8-36F5-4070-AD8D-34A44A1D4ABB",
			Name:        "TestSet07",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with higher access",
			ResourceToAccess: map[string]storage.Access{
				// Keep the access of the replaced resources which is lower
				DeploymentExtension: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "96D8F800-DACF-4FF2-8674-9AAD8230CF49",
			Name:        "TestSet08",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than one of the replaced resources",
			ResourceToAccess: map[string]storage.Access{
				// Keep the lowest access of the replaced resources
				DeploymentExtension: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "DBFF2131-811E-4F22-9386-449AF02B9053",
			Name:        "TestSet09",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_NO_ACCESS,
			},
		},
		{
			Id:          "98D88DBA-1996-40BA-BC4D-953E3D60E35A",
			Name:        "TestSet10",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				DeploymentExtension: storage.Access_NO_ACCESS,
			},
		},
		{
			Id:          "E0F50165-9914-4D0E-8C37-E8C8D482C904",
			Name:        "TestSet11",
			Description: "PermissionSet with access defined for all existing resource types",
			ResourceToAccess: map[string]storage.Access{
				// Replacing resources
				Access:              storage.Access_NO_ACCESS,
				Administration:      storage.Access_READ_ACCESS,
				Compliance:          storage.Access_NO_ACCESS,
				DeploymentExtension: storage.Access_NO_ACCESS,
				Image:               storage.Access_NO_ACCESS,
				Integration:         storage.Access_NO_ACCESS,
				// To-be-replaced-later resources
				AllComments:           storage.Access_NO_ACCESS,
				ComplianceRuns:        storage.Access_READ_WRITE_ACCESS,
				ComplianceRunSchedule: storage.Access_NO_ACCESS,
				Config:                storage.Access_READ_ACCESS,
				DebugLogs:             storage.Access_READ_WRITE_ACCESS,
				NetworkGraphConfig:    storage.Access_NO_ACCESS,
				ProbeUpload:           storage.Access_READ_WRITE_ACCESS,
				ScannerBundle:         storage.Access_NO_ACCESS,
				ScannerDefinitions:    storage.Access_READ_ACCESS,
				SensorUpgradeConfig:   storage.Access_READ_ACCESS,
				ServiceIdentity:       storage.Access_READ_ACCESS,
				// Non-replaced resources
				Alert:                            storage.Access_NO_ACCESS,
				CVE:                              storage.Access_NO_ACCESS,
				Cluster:                          storage.Access_READ_WRITE_ACCESS,
				Deployment:                       storage.Access_READ_ACCESS,
				Detection:                        storage.Access_NO_ACCESS,
				K8sRole:                          storage.Access_NO_ACCESS,
				K8sRoleBinding:                   storage.Access_READ_ACCESS,
				K8sSubject:                       storage.Access_NO_ACCESS,
				Namespace:                        storage.Access_READ_WRITE_ACCESS,
				NetworkGraph:                     storage.Access_NO_ACCESS,
				NetworkPolicy:                    storage.Access_NO_ACCESS,
				Node:                             storage.Access_READ_ACCESS,
				Policy:                           storage.Access_READ_WRITE_ACCESS,
				Role:                             storage.Access_READ_WRITE_ACCESS,
				Secret:                           storage.Access_READ_ACCESS,
				ServiceAccount:                   storage.Access_NO_ACCESS,
				VulnerabilityManagementApprovals: storage.Access_READ_WRITE_ACCESS,
				VulnerabilityManagementRequests:  storage.Access_READ_WRITE_ACCESS,
				VulnerabilityReports:             storage.Access_READ_ACCESS,
				WatchedImage:                     storage.Access_READ_ACCESS,
				// Internal resources
				ComplianceOperator: storage.Access_READ_ACCESS,
				InstallationInfo:   storage.Access_READ_ACCESS,
				Version:            storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "E79F2114-F949-411B-9F6D-4D38C1404642",
			Name:        "TestSet12",
			Description: "PermissionSet with access defined to read for all existing resource types except DebugLogs (Analyst)",
			ResourceToAccess: map[string]storage.Access{
				// Replacing resources
				Access:              storage.Access_READ_ACCESS,
				Administration:      storage.Access_READ_ACCESS,
				Compliance:          storage.Access_READ_ACCESS,
				DeploymentExtension: storage.Access_READ_ACCESS,
				Image:               storage.Access_READ_ACCESS,
				Integration:         storage.Access_READ_ACCESS,
				// To-be-replaced-later resources
				AllComments:           storage.Access_READ_ACCESS,
				ComplianceRuns:        storage.Access_READ_ACCESS,
				ComplianceRunSchedule: storage.Access_READ_ACCESS,
				Config:                storage.Access_READ_ACCESS,
				DebugLogs:             storage.Access_NO_ACCESS,
				NetworkGraphConfig:    storage.Access_READ_ACCESS,
				ProbeUpload:           storage.Access_READ_ACCESS,
				ScannerBundle:         storage.Access_READ_ACCESS,
				ScannerDefinitions:    storage.Access_READ_ACCESS,
				SensorUpgradeConfig:   storage.Access_READ_ACCESS,
				ServiceIdentity:       storage.Access_READ_ACCESS,
				// Non-replaced resources
				Alert:                            storage.Access_READ_ACCESS,
				CVE:                              storage.Access_READ_ACCESS,
				Cluster:                          storage.Access_READ_ACCESS,
				Deployment:                       storage.Access_READ_ACCESS,
				Detection:                        storage.Access_READ_ACCESS,
				K8sRole:                          storage.Access_READ_ACCESS,
				K8sRoleBinding:                   storage.Access_READ_ACCESS,
				K8sSubject:                       storage.Access_READ_ACCESS,
				Namespace:                        storage.Access_READ_ACCESS,
				NetworkGraph:                     storage.Access_READ_ACCESS,
				NetworkPolicy:                    storage.Access_READ_ACCESS,
				Node:                             storage.Access_READ_ACCESS,
				Policy:                           storage.Access_READ_ACCESS,
				Role:                             storage.Access_READ_ACCESS,
				Secret:                           storage.Access_READ_ACCESS,
				ServiceAccount:                   storage.Access_READ_ACCESS,
				VulnerabilityManagementApprovals: storage.Access_READ_ACCESS,
				VulnerabilityManagementRequests:  storage.Access_READ_ACCESS,
				VulnerabilityReports:             storage.Access_READ_ACCESS,
				WatchedImage:                     storage.Access_READ_ACCESS,
				// Internal resources
				ComplianceOperator: storage.Access_READ_ACCESS,
				InstallationInfo:   storage.Access_READ_ACCESS,
				Version:            storage.Access_READ_ACCESS,
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
		suite.Require().NoError(suite.db.Put(writeOpts, key, data))
	}

	dbs := &types.Databases{
		RocksDB: suite.db.DB,
	}

	suite.Require().NoError(migration.Run(dbs))

	var allPSsAfterMigration []*storage.PermissionSet

	for _, existing := range UnmigratedPermissionSets {
		msg, exists, err := rockshelper.ReadFromRocksDB(suite.db.DB, readOpts, &storage.PermissionSet{}, prefix, []byte(existing.GetId()))
		suite.NoError(err)
		suite.True(exists)

		allPSsAfterMigration = append(allPSsAfterMigration, msg.(*storage.PermissionSet))
	}

	expectedPSsAfterMigration := MigratedPermissionSets

	suite.ElementsMatch(expectedPSsAfterMigration, allPSsAfterMigration)
}

func (suite *psMigrationTestSuite) TestMigrationOnCleanDB() {
	dbs := &types.Databases{
		RocksDB: suite.db.DB,
	}
	suite.NoError(migration.Run(dbs))
}
