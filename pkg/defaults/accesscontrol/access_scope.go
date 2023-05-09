package accesscontrol

const (
	// UnrestrictedAccessScope is the name of the default unrestricted access scope.
	UnrestrictedAccessScope = "Unrestricted"
	// DenyAllAccessScope is the name of the default deny all access scope.
	DenyAllAccessScope = "Deny all"
)

// Postgres IDs for access scopes
// The values are UUIDs taken in descending order from ffffffff-ffff-fff4-f5ff-ffffffffffff
// Next ID: ffffffff-ffff-fff4-f5ff-fffffffffffd
const (
	unrestrictedAccessScopeID = "ffffffff-ffff-fff4-f5ff-ffffffffffff"
	denyAllAccessScopeID      = "ffffffff-ffff-fff4-f5ff-fffffffffffe"
)

// DefaultAccessScopeIDFromName returns the default access scope ID for the given name.
// In case the name is not a default access scope, return an empty string.
func DefaultAccessScopeIDFromName(name string) string {
	switch name {
	case UnrestrictedAccessScope:
		return unrestrictedAccessScopeID
	case DenyAllAccessScope:
		return denyAllAccessScopeID
	default:
		return ""
	}
}
