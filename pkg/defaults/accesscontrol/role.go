package accesscontrol

import (
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

	// NetworkGraphViewer is a role that has the minimal privileges required to display network graphs.
	NetworkGraphViewer = "Network Graph Viewer"

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
	DefaultRoleNames = set.NewStringSet(Admin, Analyst, NetworkGraphViewer, None, ContinuousIntegration, ScopeManager, SensorCreator, VulnerabilityManager, VulnMgmtApprover, VulnMgmtRequester, VulnReporter)
)

// IsDefaultRole will return true if the given role name is a default role.
func IsDefaultRole(name string) bool {
	return DefaultRoleNames.Contains(name)
}
