package accesscontrol

const (
	// UnrestrictedAccessScope is the name of the default unrestricted access scope.
	UnrestrictedAccessScope = "Unrestricted"
	// DenyAllAccessScope is the name of the default deny all access scope.
	DenyAllAccessScope = "Deny All"
)

// Postgres IDs for access scopes
// The values are UUIDs taken in descending order from ffffffff-ffff-fff4-f5ff-ffffffffffff
// Next ID: ffffffff-ffff-fff4-f5ff-fffffffffffd
const (
	unrestrictedAccessScopeID = "ffffffff-ffff-fff4-f5ff-ffffffffffff"
	denyAllAccessScopeID      = "ffffffff-ffff-fff4-f5ff-fffffffffffe"
)

var (
	// DefaultAccessScopeIDs holds all UUIDs for default access scopes.
	DefaultAccessScopeIDs = map[string]string{
		UnrestrictedAccessScope: unrestrictedAccessScopeID,
		DenyAllAccessScope:      denyAllAccessScopeID,
	}
)
