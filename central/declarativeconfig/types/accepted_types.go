package types

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
)

var (
	// AuthProviderType reflects the type of storage.AuthProvider.
	AuthProviderType = reflect.TypeFor[*storage.AuthProvider]()
	// AccessScopeType reflects the type of storage.SimpleAccessScope.
	AccessScopeType = reflect.TypeFor[*storage.SimpleAccessScope]()
	// GroupType reflects the type of storage.Group.
	GroupType = reflect.TypeFor[*storage.Group]()
	// PermissionSetType reflects the type of storage.PermissionSet.
	PermissionSetType = reflect.TypeFor[*storage.PermissionSet]()
	// RoleType reflects the type of storage.Role.
	RoleType = reflect.TypeFor[*storage.Role]()
	// NotifierType reflects the type of storage.Notifier.
	NotifierType = reflect.TypeFor[*storage.Notifier]()
	// AuthMachineToMachineConfigType reflects the type of storage.AuthMachineToMachineConfig.
	AuthMachineToMachineConfigType = reflect.TypeFor[*storage.AuthMachineToMachineConfig]()
)

// GetSupportedProtobufTypesInProcessingOrder returns the list of protobuf types
// ordered in a way that all types that can be referenced from a type in the list
// are present before the referencing type in the list. For example, Role objects
// reference a PermissionSet object and an AccessScope object, therefore
// the returned list should contain them in the [PermissionSet, AccessScope, Role]
// sequence.
func GetSupportedProtobufTypesInProcessingOrder() []reflect.Type {
	return []reflect.Type{
		AccessScopeType,
		PermissionSetType,
		RoleType,
		AuthProviderType,
		GroupType,
		NotifierType,
		AuthMachineToMachineConfigType,
	}
}
