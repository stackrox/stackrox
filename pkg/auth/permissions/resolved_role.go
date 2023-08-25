package permissions

import "github.com/stackrox/rox/generated/storage"

// ResolvedRole type unites a role with the corresponding permission set and
// access scope. It has been designed to simplify working with the new Role +
// Permission Set format but is also safe to use with the old Role only format.
//
//go:generate mockgen-wrapper
type ResolvedRole interface {
	GetRoleName() string
	GetPermissions() map[string]storage.Access
	GetAccessScope() *storage.SimpleAccessScope
}
