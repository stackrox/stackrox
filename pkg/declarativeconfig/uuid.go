package declarativeconfig

import "github.com/stackrox/rox/pkg/uuid"

const (
	authProviderUUIDNS  string = "auth-provider"
	groupUUIDNS         string = "group"
	permissionSetUUIDNS string = "permission-set"
	accessScopeUUIDNS   string = "access-scope"
)

// NewDeclarativeAuthProviderUUID creates a UUID from the name of a declarative auth provider configuration.
// The returned UUID will be deterministic.
func NewDeclarativeAuthProviderUUID(name string) uuid.UUID {
	return uuid.NewV5FromNonUUIDs(authProviderUUIDNS, name)
}

// NewDeclarativeGroupUUID creates a UUID from the name of a declarative group configuration.
// The returned UUID will be deterministic.
func NewDeclarativeGroupUUID(name string) uuid.UUID {
	return uuid.NewV5FromNonUUIDs(groupUUIDNS, name)
}

// NewDeclarativePermissionSetUUID creates a UUID from the name of a declarative permission set configuration.
// The returned UUID will be deterministic.
func NewDeclarativePermissionSetUUID(name string) uuid.UUID {
	return uuid.NewV5FromNonUUIDs(permissionSetUUIDNS, name)
}

// NewDeclarativeAccessScopeUUID creates a UUID from the name of a declarative access scope configuration.
// The returned UUID will be deterministic.
func NewDeclarativeAccessScopeUUID(name string) uuid.UUID {
	return uuid.NewV5FromNonUUIDs(accessScopeUUIDNS, name)
}
