package types

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
)

var (
	// AuthProviderType reflects the type of storage.AuthProvider.
	AuthProviderType = reflect.TypeOf((*storage.AuthProvider)(nil))
	// AccessScopeType reflects the type of storage.SimpleAccessScope.
	AccessScopeType = reflect.TypeOf((*storage.SimpleAccessScope)(nil))
	// GroupType reflects the type of storage.Group.
	GroupType = reflect.TypeOf((*storage.Group)(nil))
	// PermissionSetType reflects the type of storage.PermissionSet.
	PermissionSetType = reflect.TypeOf((*storage.PermissionSet)(nil))
	// RoleType reflects the type of storage.Role.
	RoleType = reflect.TypeOf((*storage.Role)(nil))
	// NotifierType reflects the type of storage.Notifier.
	NotifierType = reflect.TypeOf((*storage.Notifier)(nil))
	// AuthMachineToMachineConfigType reflects the type of storage.AuthMachineToMachineConfig.
	AuthMachineToMachineConfigType = reflect.TypeOf((*storage.AuthMachineToMachineConfig)(nil))
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
