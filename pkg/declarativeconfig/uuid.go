package declarativeconfig

import "github.com/stackrox/rox/pkg/uuid"

const (
	authProviderUUIDNS  string = "auth-provider"
	groupUUIDNS         string = "group"
	permissionSetUUIDNS string = "permission-set"
	accessScopeUUIDNS   string = "access-scope"
	notifierUUIDNS      string = "notifier"
	handlerUUIDNS       string = "handler"
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

// NewDeclarativeNotifierUUID creates a UUID from the name of a declarative notifier configuration.
// The returned UUID will be deterministic.
func NewDeclarativeNotifierUUID(name string) uuid.UUID {
	return uuid.NewV5FromNonUUIDs(notifierUUIDNS, name)
}

// NewDeclarativeHandlerUUID creates a UUID from the name of a declarative configuration handler.
// The returned UUID will be deterministic.
func NewDeclarativeHandlerUUID(name string) uuid.UUID {
	return uuid.NewV5FromNonUUIDs(handlerUUIDNS, name)
}
