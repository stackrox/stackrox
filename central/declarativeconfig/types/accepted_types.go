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
)
