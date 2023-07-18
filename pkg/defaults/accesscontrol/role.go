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

	// VulnMgmtApprover is a role that has the minimal privileges required to approve vulnerability deferrals or false positive requests.
	VulnMgmtApprover = "Vulnerability Management Approver"

	// VulnMgmtRequester is a role that has the minimal privileges required to request vulnerability deferrals or false positives.
	VulnMgmtRequester = "Vulnerability Management Requester"

	// TODO: ROX-14398 Remove default role VulnReporter
	// VulnReporter is a role that has the minimal privileges required to create and manage vulnerability reporting configurations.
	VulnReporter = "Vulnerability Report Creator"
)

var (
	// DefaultRoleNames is a string set containing the names of all default (built-in) Roles.
	DefaultRoleNames = set.NewStringSet(Admin, Analyst, NetworkGraphViewer, None, ContinuousIntegration, SensorCreator, VulnMgmtApprover, VulnMgmtRequester, VulnReporter)
)

// IsDefaultRole will return true if the given role name is a default role.
func IsDefaultRole(name string) bool {
	return DefaultRoleNames.Contains(name)
}
