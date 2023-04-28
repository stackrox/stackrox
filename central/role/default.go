package role

import (
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/set"
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

	// VulnerabilityManager is a role that has the necessary privileges required to view and manage system vulnerabilities and its insights.
	// This includes privileges to:
	// - view cluster, node, namespace, deployments, images (along with its scan data), and vulnerability requests.
	// - view and create requests to watch images for vulnerability insights.
	// - view and request vulnerability deferrals or false positives. This does include permissions to approve vulnerability requests.
	// - view and create vulnerability reports.
	VulnerabilityManager = "Vulnerability Manager"

	// VulnMgmtApprover is a role that has the minimal privileges required to approve vulnerability deferrals or false positive requests.
	VulnMgmtApprover = "Vulnerability Management Approver"

	// VulnMgmtRequester is a role that has the minimal privileges required to request vulnerability deferrals or false positives.
	VulnMgmtRequester = "Vulnerability Management Requester"

	// TODO: ROX-14398 Remove default role VulnReporter
	// VulnReporter is a role that has the minimal privileges required to create and manage vulnerability reporting configurations.
	VulnReporter = "Vulnerability Report Creator"

	// TODO: ROX-14398 Remove ScopeManager default role
	// ScopeManager is a role that has the minimal privileges to view and modify scopes for use in access control, vulnerability reporting etc.
	ScopeManager = "Scope Manager"
)

var (
	// DefaultRoleNames is a string set containing the names of all default (built-in) Roles.
	DefaultRoleNames = set.NewStringSet(Admin, Analyst, None, ContinuousIntegration, ScopeManager, SensorCreator, VulnerabilityManager, VulnMgmtApprover, VulnMgmtRequester, VulnReporter)

	// defaultScopesIDs is a string set containing the names of all default (built-in) scopes.
	defaultScopesIDs = set.NewFrozenStringSet(AccessScopeIncludeAll.Id, AccessScopeExcludeAll.Id)

	// AccessScopeExcludeAll has empty rules and hence excludes all
	// scoped resources. Global resources must be unaffected.
	AccessScopeExcludeAll = &storage.SimpleAccessScope{
		Id:          getAccessScopeExcludeAllID(),
		Name:        "Deny All",
		Description: "No access to scoped resources",
		Rules:       &storage.SimpleAccessScope_Rules{},
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}

	// AccessScopeIncludeAll gives access to all resources. It is checked by ID, as
	// Rules cannot represent unrestricted scope.
	AccessScopeIncludeAll = &storage.SimpleAccessScope{
		Id:          getAccessScopeIncludeAllID(),
		Name:        "Unrestricted",
		Description: "Access to all clusters and namespaces",
		Traits: &storage.Traits{
			Origin: storage.Traits_DEFAULT,
		},
	}
)

func getAccessScopeExcludeAllID() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return denyAllAccessScopeID
	}
	return EnsureValidAccessScopeID("denyall")
}

func getAccessScopeIncludeAllID() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return unrestrictedAccessScopeID
	}
	return EnsureValidAccessScopeID("unrestricted")
}

// IsDefaultRole checks if a given role corresponds to a default role.
func IsDefaultRole(role *storage.Role) bool {
	return role.GetTraits().GetOrigin() == storage.Traits_DEFAULT || DefaultRoleNames.Contains(role.GetName())
}

// IsDefaultPermissionSet checks if a given permission set corresponds to a default role.
func IsDefaultPermissionSet(permissionSet *storage.PermissionSet) bool {
	return permissionSet.GetTraits().GetOrigin() == storage.Traits_DEFAULT ||
		DefaultRoleNames.Contains(permissionSet.GetName())
}

// IsDefaultAccessScope checks if a given access scope corresponds to a
// default access scope.
func IsDefaultAccessScope(scope *storage.SimpleAccessScope) bool {
	return scope.GetTraits().GetOrigin() == storage.Traits_DEFAULT || defaultScopesIDs.Contains(scope.GetId())
}

// GetAnalystPermissions returns permissions for `Analyst` role.
func GetAnalystPermissions() []permissions.ResourceWithAccess {
	resourceToAccess := resources.AllResourcesViewPermissions()
	for i, resourceWithAccess := range resourceToAccess {
		if resourceWithAccess.Resource.GetResource() == resources.Administration.GetResource() {
			return append(resourceToAccess[:i], resourceToAccess[i+1:]...)
		}
	}
	panic("Administration resource was not found amongst all resources.")
}
