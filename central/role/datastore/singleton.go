package datastore

import (
	"github.com/stackrox/rox/central/globaldb"
	rolePkg "github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/central/role/resources"
	permissionSetStore "github.com/stackrox/rox/central/role/store/permissionset/rocksdb"
	roleStore "github.com/stackrox/rox/central/role/store/role/rocksdb"
	simpleAccessScopeStore "github.com/stackrox/rox/central/role/store/simpleaccessscope/rocksdb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/features"
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
		roleStorage, err := roleStore.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)
		permissionSetStorage, err := permissionSetStore.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)
		accessScopeStorage, err := simpleAccessScopeStore.New(globaldb.GetRocksDB())
		utils.CrashOnError(err)

		// Which role format is used is determined solely by the feature flag.
		ds = New(roleStorage, permissionSetStorage, accessScopeStorage)

		// extend default roles if vuln risk management feature flag is enabled
		if features.VulnRiskManagement.Enabled() {
			for r, a := range vulnRiskManagementDefaultRoles {
				defaultRoles[r] = a
			}
		}

		if features.VulnReporting.Enabled() {
			for r, a := range vulnReportingDefaultRoles {
				defaultRoles[r] = a
			}
		}

		roles, permissionSets, accessScopes := getDefaultObjects()
		utils.Must(roleStorage.UpsertMany(roles))
		utils.Must(permissionSetStorage.UpsertMany(permissionSets))
		utils.Must(accessScopeStorage.UpsertMany(accessScopes))
	})
	return ds
}

type roleAttributes struct {
	idSuffix           string
	description        string
	resourceWithAccess []permissions.ResourceWithAccess
}

var defaultRoles = map[string]roleAttributes{
	rolePkg.Admin: {
		idSuffix:           "admin",
		description:        "For users: use it to provide read and write access to all the resources",
		resourceWithAccess: resources.AllResourcesModifyPermissions(),
	},
	rolePkg.Analyst: {
		idSuffix:           "analyst",
		resourceWithAccess: rolePkg.GetAnalystPermissions(),
		description:        "For users: use it to give read-only access to all the resources",
	},
	rolePkg.ContinuousIntegration: {
		idSuffix:    "continuousintegration",
		description: "For automation: it includes the permissions required to enforce deployment policies",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Detection),
			permissions.Modify(resources.Image),
		},
	},
	rolePkg.None: {
		idSuffix:    "none",
		description: "For users: use it to provide no read and write access to any resource",
	},
	rolePkg.ScopeManager: {
		idSuffix:    "scopemanager",
		description: "For users: use it to create and modify scopes for the purpose of access control or vulnerability reporting",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.AuthProvider),
			permissions.View(resources.Cluster),
			permissions.View(resources.Namespace),
			permissions.View(resources.Role),
			permissions.Modify(resources.Role),
			permissions.View(resources.AuthProvider),
			permissions.View(resources.Cluster),
			permissions.View(resources.Namespace),
		},
	},
	rolePkg.SensorCreator: {
		idSuffix:    "sensorcreator",
		description: "For automation: it consists of the permissions to create Sensors in secured clusters",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.Cluster),
			permissions.Modify(resources.Cluster),
			permissions.Modify(resources.ServiceIdentity),
		},
	},
}

var vulnRiskManagementDefaultRoles = map[string]roleAttributes{
	rolePkg.VulnMgmtApprover: {
		idSuffix:    "vulnmgmtapprover",
		description: "For users: use it to provide access to approve vulnerability deferrals or false positive requests",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.VulnerabilityManagementApprovals),
			permissions.Modify(resources.VulnerabilityManagementApprovals),
		},
	},
	rolePkg.VulnMgmtRequester: {
		idSuffix:    "vulnmgmtrequester",
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
		description: "For users: use it to create and manage vulnerability reporting configurations for scheduled vulnerability reports",
		resourceWithAccess: []permissions.ResourceWithAccess{
			permissions.View(resources.VulnerabilityReports),   // required for vuln report configurations
			permissions.Modify(resources.VulnerabilityReports), // required for vuln report configurations
			permissions.View(resources.Role),                   // required for scopes
			permissions.View(resources.Image),                  // required to gather CVE data for the report
			permissions.View(resources.Notifier),               // required for vuln report configurations
			permissions.Modify(resources.Notifier),             // required for vuln report configurations

		},
	},
}

func getDefaultObjects() ([]*storage.Role, []*storage.PermissionSet, []*storage.SimpleAccessScope) {
	roles := make([]*storage.Role, 0, len(defaultRoles))
	permissionSets := make([]*storage.PermissionSet, 0, len(defaultRoles))

	for roleName, attributes := range defaultRoles {
		resourceToAccess := permissionsUtils.FromResourcesWithAccess(attributes.resourceWithAccess...)

		role := &storage.Role{
			Name:        roleName,
			Description: attributes.description,
		}

		permissionSet := &storage.PermissionSet{
			Id:               rolePkg.EnsureValidPermissionSetID(attributes.idSuffix),
			Name:             role.Name,
			Description:      role.Description,
			ResourceToAccess: resourceToAccess,
		}
		role.PermissionSetId = permissionSet.Id
		permissionSets = append(permissionSets, permissionSet)

		roles = append(roles, role)

	}
	return roles, permissionSets, []*storage.SimpleAccessScope{rolePkg.AccessScopeExcludeAll}
}
