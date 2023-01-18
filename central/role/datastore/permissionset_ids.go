package datastore

// Postgres IDs for permission sets
// The values are UUIDs taken in descending order from ffffffff-ffff-fff4-f5ff-ffffffffffff
// Next ID: ffffffff-ffff-fff4-f5ff-fffffffffff6
const (
	adminPermissionSetID                 = "ffffffff-ffff-fff4-f5ff-ffffffffffff"
	analystPermissionSetID               = "ffffffff-ffff-fff4-f5ff-fffffffffffe"
	continuousIntegrationPermissionSetID = "ffffffff-ffff-fff4-f5ff-fffffffffffd"
	nonePermissionSetID                  = "ffffffff-ffff-fff4-f5ff-fffffffffffc"
	// TODO: ROX-14398 Remove ScopeManager default role
	scopeManagerPermissionSetID      = "ffffffff-ffff-fff4-f5ff-fffffffffffb"
	sensorCreatorPermissionSetID     = "ffffffff-ffff-fff4-f5ff-fffffffffffa"
	vulnMgmtApproverPermissionSetID  = "ffffffff-ffff-fff4-f5ff-fffffffffff9"
	vulnMgmtRequesterPermissionSetID = "ffffffff-ffff-fff4-f5ff-fffffffffff8"
	vulnReporterPermissionSetID      = "ffffffff-ffff-fff4-f5ff-fffffffffff7"
)
