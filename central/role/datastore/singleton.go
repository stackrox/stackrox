package datastore

import (
	"context"

	"github.com/stackrox/rox/central/globaldb"
	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/role/store"
	PermissionSetPGStore "github.com/stackrox/rox/central/role/store/permissionset/postgres"
	permissionSetPGStore "github.com/stackrox/rox/central/role/store/permissionset/rocksdb"
	postgresRolePGStore "github.com/stackrox/rox/central/role/store/role/postgres"
	roleStore "github.com/stackrox/rox/central/role/store/role/rocksdb"
	postgresSimpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/postgres"
	simpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	ds   DataStore
	once sync.Once
)

// Singleton returns the singleton providing access to the roles store.
func Singleton() DataStore {
	once.Do(func() {
		var roleStorage store.RoleStore
		var permissionSetStorage store.PermissionSetStore
		var accessScopeStorage store.SimpleAccessScopeStore
		if env.PostgresDatastoreEnabled.BooleanSetting() {
			roleStorage = postgresRolePGStore.New(globaldb.GetPostgres())
			permissionSetStorage = PermissionSetPGStore.New(globaldb.GetPostgres())
			accessScopeStorage = postgresSimpleAccessScopeStore.New(globaldb.GetPostgres())
		} else {
			var err error
			roleStorage, err = roleStore.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
			permissionSetStorage, err = permissionSetPGStore.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
			accessScopeStorage, err = simpleAccessScopeStore.New(globaldb.GetRocksDB())
			utils.CrashOnError(err)
		}
		// Which role format is used is determined solely by the feature flag.
		ds = New(roleStorage, permissionSetStorage, accessScopeStorage)

		for r, a := range vulnReportingDefaultRoles {
			defaultRoles[r] = a
		}

		ctx := sac.WithGlobalAccessScopeChecker(context.Background(),
			sac.AllowFixedScopes(
				sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
				sac.ResourceScopeKeys(resources.Role)))
		roles, permissionSets, accessScopes := getDefaultObjects()
		utils.Must(roleStorage.UpsertMany(ctx, roles))
		utils.Must(permissionSetStorage.UpsertMany(ctx, permissionSets))
		utils.Must(accessScopeStorage.UpsertMany(ctx, accessScopes))
	})
	return ds
}

type roleAttributes struct {
	idSuffix           string
	postgresID         string // postgresID should be populated with valid UUID values.
	description        string
	resourceWithAccess []permissions.ResourceWithAccess
}

func (attributes *roleAttributes) getID() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return attributes.postgresID
	}
	return rolePkg.EnsureValidPermissionSetID(attributes.idSuffix)
}

var defaultRoles = map[string]roleAttributes{
	rolePkg.Admin: {
		idSuffix:           "admin",
		postgresID:         adminPermissionSetID,
		description:        "For users: use it to provide read and write access to all the resources",
		resourceWithAccess: resources.AllResourcesModifyPermissions(),
	},
	rolePkg.Analyst: {
		idSuffix:           "analyst",
		postgresID:         analystPermissionSetID,
		resourceWithAccess: rolePkg.GetAnalystPermissions(),
		description:        "For users: use it to give read-only access to all the resources",
	},
	rolePkg.ContinuousIntegration: {
		idSuffix:    "continuousintegration",
		postgresID:  continuousIntegrationPermissionSetID,
		description: "For automation: it includes the permissions required to enforce deployment policies",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Detection),
			permissions.Modify(resources.Image),
		},
	},
	rolePkg.None: {
		idSuffix:    "none",
		postgresID:  nonePermissionSetID,
		description: "For users: use it to provide no read and write access to any resource",
	},
	rolePkg.ScopeManager: {
		idSuffix:    "scopemanager",
		postgresID:  scopeManagerPermissionSetID,
		description: "For users: use it to create and modify scopes for the purpose of access control or vulnerability reporting",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Access),
			permissions.View(resources.Cluster),
			permissions.View(resources.Namespace),
			permissions.View(resources.Role),
			permissions.Modify(resources.Role),
		},
	},
	rolePkg.SensorCreator: {
		idSuffix:    "sensorcreator",
		postgresID:  sensorCreatorPermissionSetID,
		description: "For automation: it consists of the permissions to create Sensors in secured clusters",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Cluster),
			permissions.Modify(resources.Cluster),
			// TODO: ROX-12750 Replace ServiceIdentity with Administration.
			permissions.Modify(resources.ServiceIdentity),
		},
	},
	rolePkg.VulnMgmtApprover: {
		idSuffix:    "vulnmgmtapprover",
		postgresID:  vulnMgmtApproverPermissionSetID,
		description: "For users: use it to provide access to approve vulnerability deferrals or false positive requests",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.VulnerabilityManagementApprovals),
			permissions.Modify(resources.VulnerabilityManagementApprovals),
		},
	},
	rolePkg.VulnMgmtRequester: {
		idSuffix:    "vulnmgmtrequester",
		postgresID:  vulnMgmtRequesterPermissionSetID,
		description: "For users: use it to provide access to request vulnerability deferrals or false positives",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.VulnerabilityManagementRequests),
			permissions.Modify(resources.VulnerabilityManagementRequests),
		},
	},
}

var vulnReportingDefaultRoles = map[string]roleAttributes{
	rolePkg.VulnReporter: {
		idSuffix:    "vulnreporter",
		postgresID:  vulnReporterPermissionSetID,
		description: "For users: use it to create and manage vulnerability reporting configurations for scheduled vulnerability reports",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.WorkflowAdministration),   // required for vuln report configurations
			permissions.Modify(resources.WorkflowAdministration), // required for vuln report configurations
			permissions.View(resources.Role),                     // required for scopes
			permissions.View(resources.Image),                    // required to gather CVE data for the report
			permissions.View(resources.Integration),              // required for vuln report configurations
			permissions.View(resources.VulnerabilityReports),     // required for vuln report configurations prior to collections
			permissions.Modify(resources.VulnerabilityReports),   // required for vuln report configurations prior to collections
		},
	},
}

func getDefaultObjects() ([]*storage.Role, []*storage.PermissionSet, []*storage.SimpleAccessScope) {
	roles := make([]*storage.Role, 0, len(defaultRoles))
	permissionSets := make([]*storage.PermissionSet, 0, len(defaultRoles))

	for roleName, attributes := range defaultRoles {
		resourceToAccess := permissionsUtils.FromResourcesWithAccess(attributes.resourceWithAccess...)

		role := &storage.Role{
			Name:          roleName,
			Description:   attributes.description,
			AccessScopeId: rolePkg.AccessScopeIncludeAll.GetId(),
			Traits: &storage.Traits{
				Origin: storage.Traits_DEFAULT,
			},
		}

		permissionSet := &storage.PermissionSet{
			Id:               attributes.getID(),
			Name:             role.Name,
			Description:      role.Description,
			ResourceToAccess: resourceToAccess,
			Traits: &storage.Traits{
				Origin: storage.Traits_DEFAULT,
			},
		}
		role.PermissionSetId = permissionSet.Id
		permissionSets = append(permissionSets, permissionSet)

		roles = append(roles, role)

	}
	simpleAccessScopes := []*storage.SimpleAccessScope{
		rolePkg.AccessScopeIncludeAll,
		rolePkg.AccessScopeExcludeAll}

	return roles, permissionSets, simpleAccessScopes
}
