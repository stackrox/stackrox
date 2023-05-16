package defaults

import (
	"github.com/stackrox/rox/central/role/resources"
	rolePkg "github.com/stackrox/rox/central/role/validator"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
)

// Postgres IDs for permission sets
// The values are UUIDs taken in descending order from ffffffff-ffff-fff4-f5ff-ffffffffffff
// Next ID: ffffffff-ffff-fff4-f5ff-fffffffffff4
const (
	adminPermissionSetID                 = "ffffffff-ffff-fff4-f5ff-ffffffffffff"
	analystPermissionSetID               = "ffffffff-ffff-fff4-f5ff-fffffffffffe"
	continuousIntegrationPermissionSetID = "ffffffff-ffff-fff4-f5ff-fffffffffffd"
	nonePermissionSetID                  = "ffffffff-ffff-fff4-f5ff-fffffffffffc"
	// TODO: ROX-14398 Remove ScopeManager default role
	scopeManagerPermissionSetID       = "ffffffff-ffff-fff4-f5ff-fffffffffffb"
	sensorCreatorPermissionSetID      = "ffffffff-ffff-fff4-f5ff-fffffffffffa"
	vulnMgmtApproverPermissionSetID   = "ffffffff-ffff-fff4-f5ff-fffffffffff9"
	vulnMgmtRequesterPermissionSetID  = "ffffffff-ffff-fff4-f5ff-fffffffffff8"
	vulnReporterPermissionSetID       = "ffffffff-ffff-fff4-f5ff-fffffffffff7"
	vulnMgmtPermissionSetID           = "ffffffff-ffff-fff4-f5ff-fffffffffff6"
	networkGraphViewerPermissionSetID = "ffffffff-ffff-fff4-f5ff-fffffffffff5"
)

// Permission sets names for default roles are derived from role name. Additional permission sets for which ACS does
// not ship built-in roles must be declared in following block.
const (
	// VulnerabilityManager permission set provides necessary privileges required to view and manage system vulnerabilities and its insights.
	// This includes privileges to:
	// - view cluster, node, namespace, deployments, images (along with its scan data), and vulnerability requests.
	// - view and create requests to watch images for vulnerability insights.
	// - view and request vulnerability deferrals or false positives. This does include permissions to approve vulnerability requests.
	// - view and create vulnerability reports.
	VulnerabilityManager = "Vulnerability Manager"
)

var (
	defaultPermissionSets = map[string]permissionSetAttributes{
		Admin: {
			idSuffix:           "admin",
			postgresID:         adminPermissionSetID,
			description:        "For users: use it to provide read and write access to all the resources",
			resourceWithAccess: resources.AllResourcesModifyPermissions(),
		},
		Analyst: {
			idSuffix:           "analyst",
			postgresID:         analystPermissionSetID,
			resourceWithAccess: GetAnalystPermissions(),
			description:        "For users: use it to give read-only access to all the resources",
		},
		ContinuousIntegration: {
			idSuffix:    "continuousintegration",
			postgresID:  continuousIntegrationPermissionSetID,
			description: "For automation: it includes the permissions required to enforce deployment policies",
			resourceWithAccess: []permissions.ResourceWithAccess{
				permissions.View(resources.Detection),
				permissions.Modify(resources.Image),
			},
		},
		NetworkGraphViewer: {
			idSuffix:    "networkgraphviewer",
			postgresID:  networkGraphViewerPermissionSetID,
			description: "For users: use it to give read-only access to the NetworkGraph pages",
			resourceWithAccess: []permissions.ResourceWithAccess{
				permissions.View(resources.Deployment),
				permissions.View(resources.NetworkGraph),
				permissions.View(resources.NetworkPolicy),
			},
		},
		None: {
			idSuffix:    "none",
			postgresID:  nonePermissionSetID,
			description: "For users: use it to provide no read and write access to any resource",
		},
		// TODO: ROX-14398 Remove ScopeManager default role
		ScopeManager: {
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
		SensorCreator: {
			idSuffix:    "sensorcreator",
			postgresID:  sensorCreatorPermissionSetID,
			description: "For automation: it consists of the permissions to create Sensors in secured clusters",
			resourceWithAccess: []permissions.ResourceWithAccess{
				permissions.View(resources.Cluster),
				permissions.Modify(resources.Cluster),
				permissions.Modify(resources.Administration),
			},
		},
		VulnMgmtApprover: {
			idSuffix:    "vulnmgmtapprover",
			postgresID:  vulnMgmtApproverPermissionSetID,
			description: "For users: use it to provide access to approve vulnerability deferrals or false positive requests",
			resourceWithAccess: []permissions.ResourceWithAccess{
				permissions.View(resources.VulnerabilityManagementApprovals),
				permissions.Modify(resources.VulnerabilityManagementApprovals),
			},
		},
		VulnMgmtRequester: {
			idSuffix:    "vulnmgmtrequester",
			postgresID:  vulnMgmtRequesterPermissionSetID,
			description: "For users: use it to provide access to request vulnerability deferrals or false positives",
			resourceWithAccess: []permissions.ResourceWithAccess{
				permissions.View(resources.VulnerabilityManagementRequests),
				permissions.Modify(resources.VulnerabilityManagementRequests),
			},
		},
		// TODO ROX-13888 when we migrate to WorkflowAdministration we can remove VulnerabilityReports and Role resources
		VulnReporter: {
			idSuffix:    "vulnreporter",
			postgresID:  vulnReporterPermissionSetID,
			description: "For users: use it to create and manage vulnerability reporting configurations for scheduled vulnerability reports",
			resourceWithAccess: func() []permissions.ResourceWithAccess {
				if !env.PostgresDatastoreEnabled.BooleanSetting() {
					return []permissions.ResourceWithAccess{
						permissions.View(resources.Role),                   // required for scopes
						permissions.View(resources.Integration),            // required for vuln report configurations
						permissions.View(resources.VulnerabilityReports),   // required for vuln report configurations prior to collections
						permissions.Modify(resources.VulnerabilityReports), // required for vuln report configurations prior to collections
					}
				}
				return []permissions.ResourceWithAccess{
					permissions.View(resources.WorkflowAdministration),   // required for vuln report configurations
					permissions.Modify(resources.WorkflowAdministration), // required for vuln report configurations
					permissions.View(resources.Integration),              // required for vuln report configurations
				}
			}(),
		},
		VulnerabilityManager: {
			idSuffix:    "vulnmgmt",
			postgresID:  vulnMgmtPermissionSetID,
			description: "For users: use it to provide access to analyze and manage system vulnerabilities",
			resourceWithAccess: []permissions.ResourceWithAccess{
				permissions.View(resources.Cluster),
				permissions.View(resources.Node),
				permissions.View(resources.Namespace),
				permissions.View(resources.Deployment),
				permissions.View(resources.Image),
				permissions.View(resources.Integration),
				permissions.Modify(resources.WatchedImage),
				permissions.Modify(resources.VulnerabilityManagementRequests),
				permissions.Modify(resources.VulnerabilityReports),
				permissions.Modify(resources.WorkflowAdministration),
			},
		},
	}
)

type permissionSetAttributes struct {
	idSuffix           string
	postgresID         string // postgresID should be populated with valid UUID values.
	description        string
	resourceWithAccess []permissions.ResourceWithAccess
}

func (attributes *permissionSetAttributes) getID() string {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return attributes.postgresID
	}
	return rolePkg.EnsureValidPermissionSetID(attributes.idSuffix)
}
