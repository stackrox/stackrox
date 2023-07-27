//go:build sql_integration

package m177tom178

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/frozenschema/v73"
	permissionsetpostgresstore "github.com/stackrox/rox/migrator/migrations/m_177_to_m_178_group_permissions/permissionsetpostgresstore"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

// Non-replaced resources
const (
	Access                           = "Access"
	Alert                            = "Alert"
	CVE                              = "CVE"
	Cluster                          = "Cluster"
	Deployment                       = "Deployment"
	DeploymentExtension              = "DeploymentExtension"
	Detection                        = "Detection"
	K8sRole                          = "K8sRole"
	K8sRoleBinding                   = "K8sRoleBinding"
	K8sSubject                       = "K8sSubject"
	Image                            = "Image"
	Integration                      = "Integration"
	Namespace                        = "Namespace"
	NetworkGraph                     = "NetworkGraph"
	NetworkPolicy                    = "NetworkPolicy"
	Node                             = "Node"
	Secret                           = "Secret"
	ServiceAccount                   = "ServiceAccount"
	VulnerabilityManagementApprovals = "VulnerabilityManagementApprovals"
	VulnerabilityManagementRequests  = "VulnerabilityManagementRequests"
	WatchedImage                     = "WatchedImage"
	// Non-replaced internal resources
	ComplianceOperator = "ComplianceOperator"
	InstallationInfo   = "InstallationInfo"
	Version            = "Version"
	// To be replaced later resources
	Policy               = "Policy"
	Role                 = "Role"
	VulnerabilityReports = "VulnerabilityReports"
)

var (
	ctx = sac.WithAllAccess(context.Background())

	unmigratedPermissionSets = []*storage.PermissionSet{
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
				ComplianceRuns: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "97A38C2D-D11D-4355-AD80-732F3661EC4B",
			Name:        "TestSet03",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with lower access",
			ResourceToAccess: map[string]storage.Access{
				Administration: storage.Access_NO_ACCESS,
				Config:         storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "7035AD8F-E811-484B-AE36-E5877325B3F0",
			Name:        "TestSet04",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				Administration: storage.Access_READ_ACCESS,
				ProbeUpload:    storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "589ADE2F-BD33-4BA7-9821-3818832C5A79",
			Name:        "TestSet05",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				Administration:      storage.Access_READ_WRITE_ACCESS,
				SensorUpgradeConfig: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "A78B24E1-F0ED-456A-B679-BADF4C47F654",
			Name:        "TestSet06",
			Description: "PermissionSet with two replaced resources for which the replacement resource is not yet set",
			ResourceToAccess: map[string]storage.Access{
				AllComments: storage.Access_READ_WRITE_ACCESS,
				DebugLogs:   storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "C0EE37B8-36F5-4070-AD8D-34A44A1D4ABB",
			Name:        "TestSet07",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with higher access",
			ResourceToAccess: map[string]storage.Access{
				Administration: storage.Access_READ_WRITE_ACCESS,
				ProbeUpload:    storage.Access_READ_ACCESS,
				ScannerBundle:  storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "96D8F800-DACF-4FF2-8674-9AAD8230CF49",
			Name:        "TestSet08",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than one of the replaced resources",
			ResourceToAccess: map[string]storage.Access{
				Administration:      storage.Access_READ_ACCESS,
				ScannerBundle:       storage.Access_READ_ACCESS,
				SensorUpgradeConfig: storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			Id:          "DBFF2131-811E-4F22-9386-449AF02B9053",
			Name:        "TestSet09",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				Administration: storage.Access_NO_ACCESS,
				DebugLogs:      storage.Access_READ_ACCESS,
				ProbeUpload:    storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "98D88DBA-1996-40BA-BC4D-953E3D60E35A",
			Name:        "TestSet10",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				Administration:     storage.Access_NO_ACCESS,
				NetworkGraphConfig: storage.Access_READ_WRITE_ACCESS,
				ServiceIdentity:    storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "E0F50165-9914-4D0E-8C37-E8C8D482C904",
			Name:        "TestSet11",
			Description: "PermissionSet with access defined for all existing resource types",
			ResourceToAccess: map[string]storage.Access{
				// Replacing resources
				Administration: storage.Access_READ_ACCESS,
				Compliance:     storage.Access_NO_ACCESS,
				// Replaced resources
				AllComments:         storage.Access_NO_ACCESS,
				ComplianceRuns:      storage.Access_READ_WRITE_ACCESS,
				Config:              storage.Access_READ_ACCESS,
				DebugLogs:           storage.Access_READ_WRITE_ACCESS,
				NetworkGraphConfig:  storage.Access_NO_ACCESS,
				ProbeUpload:         storage.Access_READ_WRITE_ACCESS,
				ScannerBundle:       storage.Access_NO_ACCESS,
				ScannerDefinitions:  storage.Access_READ_ACCESS,
				SensorUpgradeConfig: storage.Access_READ_ACCESS,
				ServiceIdentity:     storage.Access_READ_ACCESS,
				// Non-replaced resources
				Access:                           storage.Access_READ_ACCESS,
				Alert:                            storage.Access_NO_ACCESS,
				CVE:                              storage.Access_NO_ACCESS,
				Cluster:                          storage.Access_READ_WRITE_ACCESS,
				Deployment:                       storage.Access_READ_ACCESS,
				DeploymentExtension:              storage.Access_NO_ACCESS,
				Detection:                        storage.Access_NO_ACCESS,
				Image:                            storage.Access_NO_ACCESS,
				Integration:                      storage.Access_READ_WRITE_ACCESS,
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
				Administration: storage.Access_READ_ACCESS,
				Compliance:     storage.Access_READ_ACCESS,
				// Replaced resources
				AllComments:         storage.Access_READ_ACCESS,
				ComplianceRuns:      storage.Access_READ_ACCESS,
				Config:              storage.Access_READ_ACCESS,
				DebugLogs:           storage.Access_READ_WRITE_ACCESS,
				NetworkGraphConfig:  storage.Access_READ_ACCESS,
				ProbeUpload:         storage.Access_READ_ACCESS,
				ScannerBundle:       storage.Access_READ_ACCESS,
				ScannerDefinitions:  storage.Access_READ_ACCESS,
				SensorUpgradeConfig: storage.Access_READ_ACCESS,
				ServiceIdentity:     storage.Access_READ_ACCESS,
				// Non-replaced resources
				Access:                           storage.Access_READ_ACCESS,
				Alert:                            storage.Access_READ_ACCESS,
				CVE:                              storage.Access_READ_ACCESS,
				Cluster:                          storage.Access_READ_ACCESS,
				Deployment:                       storage.Access_READ_ACCESS,
				DeploymentExtension:              storage.Access_READ_ACCESS,
				Detection:                        storage.Access_READ_ACCESS,
				Image:                            storage.Access_READ_ACCESS,
				Integration:                      storage.Access_READ_ACCESS,
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

	migratedPermissionSets = []*storage.PermissionSet{
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
				Compliance: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "97A38C2D-D11D-4355-AD80-732F3661EC4B",
			Name:        "TestSet03",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with lower access",
			ResourceToAccess: map[string]storage.Access{
				// Keeps the access of the replacing resource which is lower
				Administration: storage.Access_NO_ACCESS,
			},
		},
		{
			Id:          "7035AD8F-E811-484B-AE36-E5877325B3F0",
			Name:        "TestSet04",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				Administration: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "589ADE2F-BD33-4BA7-9821-3818832C5A79",
			Name:        "TestSet05",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccess: map[string]storage.Access{
				// Keep the access of the replaced resource which is lower
				Administration: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "A78B24E1-F0ED-456A-B679-BADF4C47F654",
			Name:        "TestSet06",
			Description: "PermissionSet with two replaced resources for which the replacement resource is not yet set",
			ResourceToAccess: map[string]storage.Access{
				// Keep the lowest access of the replaced resources
				Administration: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "C0EE37B8-36F5-4070-AD8D-34A44A1D4ABB",
			Name:        "TestSet07",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with higher access",
			ResourceToAccess: map[string]storage.Access{
				// Keep the access of the replaced resources which is lower
				Administration: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "96D8F800-DACF-4FF2-8674-9AAD8230CF49",
			Name:        "TestSet08",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than one of the replaced resources",
			ResourceToAccess: map[string]storage.Access{
				// Keep the lowest access of the replaced resources
				Administration: storage.Access_READ_ACCESS,
			},
		},
		{
			Id:          "DBFF2131-811E-4F22-9386-449AF02B9053",
			Name:        "TestSet09",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				Administration: storage.Access_NO_ACCESS,
			},
		},
		{
			Id:          "98D88DBA-1996-40BA-BC4D-953E3D60E35A",
			Name:        "TestSet10",
			Description: "PermissionSet with two replaced resources for which the replacement resource is set with access lower than both replaced resources",
			ResourceToAccess: map[string]storage.Access{
				Administration: storage.Access_NO_ACCESS,
			},
		},
		{
			Id:          "E0F50165-9914-4D0E-8C37-E8C8D482C904",
			Name:        "TestSet11",
			Description: "PermissionSet with access defined for all existing resource types",
			ResourceToAccess: map[string]storage.Access{
				// Replacing resources
				Administration: storage.Access_NO_ACCESS,
				Compliance:     storage.Access_NO_ACCESS,
				// Non-replaced resources
				Access:                           storage.Access_READ_ACCESS,
				Alert:                            storage.Access_NO_ACCESS,
				CVE:                              storage.Access_NO_ACCESS,
				Cluster:                          storage.Access_READ_WRITE_ACCESS,
				Deployment:                       storage.Access_READ_ACCESS,
				DeploymentExtension:              storage.Access_NO_ACCESS,
				Detection:                        storage.Access_NO_ACCESS,
				Image:                            storage.Access_NO_ACCESS,
				Integration:                      storage.Access_READ_WRITE_ACCESS,
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
				Administration: storage.Access_READ_ACCESS,
				Compliance:     storage.Access_READ_ACCESS,
				// Non-replaced resources
				Access:                           storage.Access_READ_ACCESS,
				Alert:                            storage.Access_READ_ACCESS,
				CVE:                              storage.Access_READ_ACCESS,
				Cluster:                          storage.Access_READ_ACCESS,
				Deployment:                       storage.Access_READ_ACCESS,
				DeploymentExtension:              storage.Access_READ_ACCESS,
				Detection:                        storage.Access_READ_ACCESS,
				Image:                            storage.Access_READ_ACCESS,
				Integration:                      storage.Access_READ_ACCESS,
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

	db *pghelper.TestPostgres
}

func TestMigration(t *testing.T) {
	suite.Run(t, new(psMigrationTestSuite))
}

func (s *psMigrationTestSuite) SetupSuite() {
	s.db = pghelper.ForT(s.T(), false)
	pgutils.CreateTableFromModel(ctx, s.db.GetGormDB(), frozenSchema.CreateTablePermissionSetsStmt)
}

func (s *psMigrationTestSuite) TearDownSuite() {
	s.db.Teardown(s.T())
}

func (s *psMigrationTestSuite) TestMigration() {
	store := permissionsetpostgresstore.New(s.db.DB)

	s.Require().NoError(store.UpsertMany(ctx, unmigratedPermissionSets))

	dbs := &types.Databases{
		PostgresDB: s.db.DB,
	}

	s.Require().NoError(migration.Run(dbs))

	allPSAfterMigration := make([]*storage.PermissionSet, 0, len(unmigratedPermissionSets))
	s.NoError(store.Walk(ctx, func(obj *storage.PermissionSet) error {
		allPSAfterMigration = append(allPSAfterMigration, obj)
		return nil
	}))

	s.ElementsMatch(migratedPermissionSets, allPSAfterMigration)
}
