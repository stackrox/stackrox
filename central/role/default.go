package role

import (
	"github.com/stackrox/stackrox/central/role/resources"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
	"github.com/stackrox/stackrox/pkg/grpc/authn"
	"github.com/stackrox/stackrox/pkg/set"
)

// All builtin, immutable role names are declared in the block below.
const (
	// Admin is a role that's, well, authorized to do anything.
	Admin = "Admin"

	// Analyst is a role that has read access to all resources.
	Analyst = "Analyst"

	// None role has no access.
	None = authn.NoneRole

	// ContinuousIntegration is for CI pipelines.
	ContinuousIntegration = "Continuous Integration"

	// SensorCreator is a role that has the minimal privileges required to create a sensor.
	SensorCreator = "Sensor Creator"

	// VulnMgmtApprover is a role that has the minimal privileges required to approve vulnerability deferrals or false positive requests.
	VulnMgmtApprover = "Vulnerability Management Approver"

	// VulnMgmtRequester is a role that has the minimal privileges required to request vulnerability deferrals or false positives.
	VulnMgmtRequester = "Vulnerability Management Requester"

	// VulnReporter is a role that has the minimal privileges required to create and manage vulnerability reporting configurations.
	VulnReporter = "Vulnerability Report Creator"

	// ScopeManager is a role that has the minimal privileges to view and modify scopes for use in access control, vulnerability reporting etc.
	ScopeManager = "Scope Manager"
)

var (
	// DefaultRoleNames is a string set containing the names of all default (built-in) Roles.
	DefaultRoleNames = set.NewStringSet(Admin, Analyst, None, ContinuousIntegration, ScopeManager, SensorCreator, VulnMgmtApprover, VulnMgmtRequester, VulnReporter)

	// defaultScopesIDs is a string set containing the names of all default (built-in) scopes.
	defaultScopesIDs = set.NewFrozenStringSet(AccessScopeIncludeAll.Id, AccessScopeExcludeAll.Id)

	// AccessScopeExcludeAll has empty rules and hence excludes all
	// scoped resources. Global resources must be unaffected.
	AccessScopeExcludeAll = &storage.SimpleAccessScope{
		Id:          EnsureValidAccessScopeID("denyall"),
		Name:        "Deny All",
		Description: "No access to scoped resources",
		Rules:       &storage.SimpleAccessScope_Rules{},
	}

	// AccessScopeIncludeAll gives access to all resources. It is checked by ID, as
	// Rules cannot represent unrestricted scope.
	AccessScopeIncludeAll = &storage.SimpleAccessScope{
		Id:          EnsureValidAccessScopeID("unrestricted"),
		Name:        "Unrestricted",
		Description: "Access to all clusters and namespaces",
	}
)

// IsDefaultRoleName checks if a given role name corresponds to a default role.
func IsDefaultRoleName(name string) bool {
	return DefaultRoleNames.Contains(name)
}

// IsDefaultAccessScope checks if a given access scope id corresponds to a
// default access scope.
func IsDefaultAccessScope(id string) bool {
	return defaultScopesIDs.Contains(id)
}

// GetAnalystPermissions returns permissions for `Analyst` role.
func GetAnalystPermissions() []permissions.ResourceWithAccess {
	resourceToAccess := resources.AllResourcesViewPermissions()
	for i, resourceWithAccess := range resourceToAccess {
		if resourceWithAccess.Resource.GetResource() == resources.DebugLogs.GetResource() {
			return append(resourceToAccess[:i], resourceToAccess[i+1:]...)
		}
	}
	panic("DebugLogs resource was not found amongst all resources.")
}
