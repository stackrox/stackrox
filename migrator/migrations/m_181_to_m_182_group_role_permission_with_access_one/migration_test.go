//go:build sql_integration

package m181tom182

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	permissionsetpostgresstore "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_group_role_permission_with_access_one/permissionsetstore"
	frozenSchema "github.com/stackrox/rox/migrator/migrations/m_181_to_m_182_group_role_permission_with_access_one/schema"
	pghelper "github.com/stackrox/rox/migrator/migrations/postgreshelper"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
)

// Non-replaced resources
const (
	Administration                   = "Administration"
	Alert                            = "Alert"
	CVE                              = "CVE"
	Cluster                          = "Cluster"
	Compliance                       = "Compliance"
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
	VulnerabilityReports = "VulnerabilityReports"
)

type testItem struct {
	ID                            string
	Name                          string
	Description                   string
	ResourceToAccessPreMigration  map[string]storage.Access
	ResourceToAccessPostMigration map[string]storage.Access
	Traits                        *storage.Traits
}

func GetPreMigrationPermissionSet(item *testItem) *storage.PermissionSet {
	return &storage.PermissionSet{
		Id:               item.ID,
		Name:             item.Name,
		Description:      item.Description,
		ResourceToAccess: item.ResourceToAccessPreMigration,
		Traits:           item.Traits,
	}
}

func GetPostMigrationPermissionSet(item *testItem) *storage.PermissionSet {
	return &storage.PermissionSet{
		Id:               item.ID,
		Name:             item.Name,
		Description:      item.Description,
		ResourceToAccess: item.ResourceToAccessPostMigration,
		Traits:           item.Traits,
	}
}

func GetDataSetPreMigration() []*storage.PermissionSet {
	res := make([]*storage.PermissionSet, 0, len(testData))
	for _, item := range testData {
		res = append(res, GetPreMigrationPermissionSet(item))
	}
	return res
}

func GetDataSetPostMigration() []*storage.PermissionSet {
	res := make([]*storage.PermissionSet, 0, len(testData))
	for _, item := range testData {
		res = append(res, GetPostMigrationPermissionSet(item))
	}
	return res
}

var (
	ctx = sac.WithAllAccess(context.Background())

	testData = []*testItem{
		{
			ID:          "AA4618AC-EDD7-4756-828F-FA8424DE138E",
			Name:        "TestSet01",
			Description: "PermissionSet with no resource that requires replacement",
			ResourceToAccessPreMigration: map[string]storage.Access{
				Administration: storage.Access_READ_ACCESS,
				Alert:          storage.Access_READ_WRITE_ACCESS,
			},
			ResourceToAccessPostMigration: map[string]storage.Access{
				Administration: storage.Access_READ_ACCESS,
				Alert:          storage.Access_READ_WRITE_ACCESS,
			},
		},
		{
			ID:          "6C618B1C-8919-4939-8A90-082EC9A90DA4",
			Name:        "TestSet02",
			Description: "PermissionSet with a replaced resource for which the replacement resource is not yet set",
			ResourceToAccessPreMigration: map[string]storage.Access{
				Role: storage.Access_READ_ACCESS,
			},
			ResourceToAccessPostMigration: map[string]storage.Access{
				Access: storage.Access_READ_ACCESS,
			},
		},
		{
			ID:          "97A38C2D-D11D-4355-AD80-732F3661EC4B",
			Name:        "TestSet03",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with lower access",
			ResourceToAccessPreMigration: map[string]storage.Access{
				Access: storage.Access_NO_ACCESS,
				Role:   storage.Access_READ_ACCESS,
			},
			ResourceToAccessPostMigration: map[string]storage.Access{
				// Keeps the access of the replacing resource which is lower
				Access: storage.Access_NO_ACCESS,
			},
		},
		{
			ID:          "7035AD8F-E811-484B-AE36-E5877325B3F0",
			Name:        "TestSet04",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with same access",
			ResourceToAccessPreMigration: map[string]storage.Access{
				Access: storage.Access_READ_ACCESS,
				Role:   storage.Access_READ_ACCESS,
			},
			ResourceToAccessPostMigration: map[string]storage.Access{
				Access: storage.Access_READ_ACCESS,
			},
		},
		{
			ID:          "589ADE2F-BD33-4BA7-9821-3818832C5A79",
			Name:        "TestSet05",
			Description: "PermissionSet with a replaced resource for which the replacement resource is set with higher access",
			ResourceToAccessPreMigration: map[string]storage.Access{
				Access: storage.Access_READ_WRITE_ACCESS,
				Role:   storage.Access_READ_ACCESS,
			},
			ResourceToAccessPostMigration: map[string]storage.Access{
				// Keep the access of the replaced resource which is lower
				Access: storage.Access_READ_ACCESS,
			},
		},

		{
			ID:          "E0F50165-9914-4D0E-8C37-E8C8D482C904",
			Name:        "TestSet11",
			Description: "PermissionSet with access defined for all existing resource types",
			ResourceToAccessPreMigration: map[string]storage.Access{
				// Replacing resources
				Access: storage.Access_READ_ACCESS,
				// Replaced resources
				Role: storage.Access_READ_WRITE_ACCESS,
				// Non-replaced resources
				Administration:                   storage.Access_READ_ACCESS,
				Alert:                            storage.Access_NO_ACCESS,
				CVE:                              storage.Access_NO_ACCESS,
				Cluster:                          storage.Access_READ_WRITE_ACCESS,
				Compliance:                       storage.Access_NO_ACCESS,
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
			ResourceToAccessPostMigration: map[string]storage.Access{
				// Replacing resources
				Access: storage.Access_READ_ACCESS,
				// Non-replaced resources
				Administration:                   storage.Access_READ_ACCESS,
				Alert:                            storage.Access_NO_ACCESS,
				CVE:                              storage.Access_NO_ACCESS,
				Cluster:                          storage.Access_READ_WRITE_ACCESS,
				Compliance:                       storage.Access_NO_ACCESS,
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
			ID:          "E79F2114-F949-411B-9F6D-4D38C1404642",
			Name:        "TestSet12",
			Description: "PermissionSet with access defined to read for all existing resource types except DebugLogs (Analyst)",
			ResourceToAccessPreMigration: map[string]storage.Access{
				// Replacing resources
				Access: storage.Access_READ_ACCESS,
				// Replaced resources
				Role: storage.Access_READ_ACCESS,
				// Non-replaced resources
				Administration:                   storage.Access_READ_ACCESS,
				Alert:                            storage.Access_READ_ACCESS,
				CVE:                              storage.Access_READ_ACCESS,
				Cluster:                          storage.Access_READ_ACCESS,
				Compliance:                       storage.Access_READ_ACCESS,
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
			ResourceToAccessPostMigration: map[string]storage.Access{
				// Replacing resources
				Access: storage.Access_READ_ACCESS,
				// Non-replaced resources
				Administration:                   storage.Access_READ_ACCESS,
				Alert:                            storage.Access_READ_ACCESS,
				CVE:                              storage.Access_READ_ACCESS,
				Cluster:                          storage.Access_READ_ACCESS,
				Compliance:                       storage.Access_READ_ACCESS,
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
		{
			ID:          "ffffffff-ffff-fff4-f5ff-fffffffffffb",
			Name:        "Scope Manager",
			Description: "For users: use it to create and modify scopes for the purpose of access control or vulnerability reporting",
			ResourceToAccessPreMigration: map[string]storage.Access{
				Access:    storage.Access_READ_ACCESS,
				Cluster:   storage.Access_READ_ACCESS,
				Namespace: storage.Access_READ_ACCESS,
				Role:      storage.Access_READ_WRITE_ACCESS,
			},
			ResourceToAccessPostMigration: map[string]storage.Access{
				Access:    storage.Access_READ_ACCESS,
				Cluster:   storage.Access_READ_ACCESS,
				Namespace: storage.Access_READ_ACCESS,
			},
			Traits: &storage.Traits{
				Origin: storage.Traits_DEFAULT,
			},
		},
	}

	unmigratedPermissionSets = GetDataSetPreMigration()

	migratedPermissionSets = GetDataSetPostMigration()
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
