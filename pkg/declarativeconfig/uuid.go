package declarativeconfig

import "github.com/stackrox/rox/pkg/uuid"

const (
	authProviderNS  string = "auth-provider"
	groupNS         string = "group"
	permissionSetNS string = "permission-set"
	accessScopeNS   string = "access-scope"
)

// NewDeclarativeAuthProviderUUID creates a UUID from the name of a declarative auth provider configuration.
// The returned UUID will be deterministic.
func NewDeclarativeAuthProviderUUID(name string) uuid.UUID {
	return uuid.NewV5FromNonUUIDs(authProviderNS, name)
}

// NewDeclarativeGroupUUID creates a UUID from the name of a declarative group configuration.
// The returned UUID will be deterministic.
func NewDeclarativeGroupUUID(name string) uuid.UUID {
	return uuid.NewV5FromNonUUIDs(groupNS, name)
}

// NewDeclarativePermissionSetUUID creates a UUID from the name of a declarative permission set configuration.
// The returned UUID will be deterministic.
func NewDeclarativePermissionSetUUID(name string) uuid.UUID {
	return uuid.NewV5FromNonUUIDs(permissionSetNS, name)
}

// NewDeclarativeAccessScopeUUID creates a UUID from the name of a declarative access scope configuration.
// The returned UUID will be deterministic.
func NewDeclarativeAccessScopeUUID(name string) uuid.UUID {
	return uuid.NewV5FromNonUUIDs(accessScopeNS, name)
}
