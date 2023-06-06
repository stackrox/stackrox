package accesscontrol

// Postgres IDs for permission sets
// The values are UUIDs taken in descending order from ffffffff-ffff-fff4-f5ff-ffffffffffff
// Next ID: ffffffff-ffff-fff4-f5ff-fffffffffff3
const (
	adminPermissionSetID                 = "ffffffff-ffff-fff4-f5ff-ffffffffffff"
	analystPermissionSetID               = "ffffffff-ffff-fff4-f5ff-fffffffffffe"
	continuousIntegrationPermissionSetID = "ffffffff-ffff-fff4-f5ff-fffffffffffd"
	nonePermissionSetID                  = "ffffffff-ffff-fff4-f5ff-fffffffffffc"
	// DO NOT RE-USE "ffffffff-ffff-fff4-f5ff-fffffffffffb"
	// the ID was used for the ScopeManager default permission set, and may not have been removed by migration (182 to 183).
	sensorCreatorPermissionSetID      = "ffffffff-ffff-fff4-f5ff-fffffffffffa"
	vulnMgmtApproverPermissionSetID   = "ffffffff-ffff-fff4-f5ff-fffffffffff9"
	vulnMgmtRequesterPermissionSetID  = "ffffffff-ffff-fff4-f5ff-fffffffffff8"
	vulnReporterPermissionSetID       = "ffffffff-ffff-fff4-f5ff-fffffffffff7"
	vulnMgmtConsumerPermissionSetID   = "ffffffff-ffff-fff4-f5ff-fffffffffff6"
	networkGraphViewerPermissionSetID = "ffffffff-ffff-fff4-f5ff-fffffffffff5"
	vulnMgmtAdminPermissionSetID      = "ffffffff-ffff-fff4-f5ff-fffffffffff4"
)

const (
	// VulnerabilityManagementConsumer permission set provides necessary privileges required to view system vulnerabilities and its insights.
	// This includes privileges to:
	// - view node, deployments, images (along with its scan data), and vulnerability requests.
	// - view watched images along with its scan data.
	// - view and request vulnerability deferrals or false positives. This does include permissions to approve vulnerability requests.
	// - view vulnerability report configurations.
	VulnerabilityManagementConsumer = "Vulnerability Management Consumer"

	// VulnerabilityManagementAdmin permission set provides necessary privileges required to view and manage system vulnerabilities and its insights.
	// This includes privileges to:
	// - view cluster, node, namespace, deployments, images (along with its scan data), and vulnerability requests.
	// - view and create requests to watch images.
	// - view, request, and approve/deny vulnerability deferrals or false positives.
	// - view and create vulnerability report configurations.
	VulnerabilityManagementAdmin = "Vulnerability Management Admin"
)

var (
	// DefaultPermissionSetIDs is a list of all permission set IDs keyed by their name.
	DefaultPermissionSetIDs = map[string]string{
		Admin:                           adminPermissionSetID,
		Analyst:                         analystPermissionSetID,
		ContinuousIntegration:           continuousIntegrationPermissionSetID,
		NetworkGraphViewer:              networkGraphViewerPermissionSetID,
		None:                            nonePermissionSetID,
		SensorCreator:                   sensorCreatorPermissionSetID,
		VulnMgmtApprover:                vulnMgmtApproverPermissionSetID,
		VulnMgmtRequester:               vulnMgmtRequesterPermissionSetID,
		VulnReporter:                    vulnReporterPermissionSetID,
		VulnerabilityManagementConsumer: vulnMgmtConsumerPermissionSetID,
		VulnerabilityManagementAdmin:    vulnMgmtAdminPermissionSetID,
	}
)

// IsDefaultPermissionSet will return true if the given permission set name is a default permission set.
func IsDefaultPermissionSet(name string) bool {
	return DefaultRoleNames.Contains(name)
}
