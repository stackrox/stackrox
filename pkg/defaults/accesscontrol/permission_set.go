package accesscontrol

// Postgres IDs for permission sets
// The values are UUIDs taken in descending order from ffffffff-ffff-fff4-f5ff-ffffffffffff
// Next ID: ffffffff-ffff-fff4-f5ff-fffffffffff4
const (
	adminPermissionSetID                 = "ffffffff-ffff-fff4-f5ff-ffffffffffff"
	analystPermissionSetID               = "ffffffff-ffff-fff4-f5ff-fffffffffffe"
	continuousIntegrationPermissionSetID = "ffffffff-ffff-fff4-f5ff-fffffffffffd"
	nonePermissionSetID                  = "ffffffff-ffff-fff4-f5ff-fffffffffffc"
	// "ffffffff-ffff-fff4-f5ff-fffffffffffb" was used for the ScopeManager default permission set,
	// references can remain after the removal.
	sensorCreatorPermissionSetID      = "ffffffff-ffff-fff4-f5ff-fffffffffffa"
	vulnMgmtApproverPermissionSetID   = "ffffffff-ffff-fff4-f5ff-fffffffffff9"
	vulnMgmtRequesterPermissionSetID  = "ffffffff-ffff-fff4-f5ff-fffffffffff8"
	vulnReporterPermissionSetID       = "ffffffff-ffff-fff4-f5ff-fffffffffff7"
	vulnMgmtPermissionSetID           = "ffffffff-ffff-fff4-f5ff-fffffffffff6"
	networkGraphViewerPermissionSetID = "ffffffff-ffff-fff4-f5ff-fffffffffff5"
)

var (
	// DefaultPermissionSetIDs is a list of all permission set IDs keyed by their name.
	DefaultPermissionSetIDs = map[string]string{
		Admin:                 adminPermissionSetID,
		Analyst:               analystPermissionSetID,
		ContinuousIntegration: continuousIntegrationPermissionSetID,
		NetworkGraphViewer:    networkGraphViewerPermissionSetID,
		None:                  nonePermissionSetID,
		SensorCreator:         sensorCreatorPermissionSetID,
		VulnMgmtApprover:      vulnMgmtApproverPermissionSetID,
		VulnMgmtRequester:     vulnMgmtRequesterPermissionSetID,
		VulnReporter:          vulnReporterPermissionSetID,
		VulnerabilityManager:  vulnMgmtPermissionSetID,
	}
)

// IsDefaultPermissionSet will return true if the given permission set name is a default permission set.
func IsDefaultPermissionSet(name string) bool {
	return DefaultRoleNames.Contains(name)
}
