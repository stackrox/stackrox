package types

import (
	"reflect"

	"github.com/stackrox/rox/generated/storage"
)

var (
	// AuthProviderType ...
	AuthProviderType = reflect.TypeOf((*storage.AuthProvider)(nil))
	// AccessScopeType ...
	AccessScopeType = reflect.TypeOf((*storage.SimpleAccessScope)(nil))
	// GroupType ...
	GroupType = reflect.TypeOf((*storage.Group)(nil))
	// PermissionSetType ...
	PermissionSetType = reflect.TypeOf((*storage.PermissionSet)(nil))
	// RoleType ...
	RoleType = reflect.TypeOf((*storage.Role)(nil))
)
