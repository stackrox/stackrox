package role

// Postgres IDs for access scopes
// The values are UUIDs taken in descending order from ffffffff-ffff-fff4-f5ff-ffffffffffff
// Next ID: ffffffff-ffff-fff4-f5ff-fffffffffffd
const (
	denyAllAccessScopeID      = "ffffffff-ffff-fff4-f5ff-fffffffffffe"
	unrestrictedAccessScopeID = "ffffffff-ffff-fff4-f5ff-ffffffffffff"
)
