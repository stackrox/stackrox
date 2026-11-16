package mapper

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

// internalRolesAttributeKey is the UserDescriptor attribute key that holds
// the JSON-serialized internal roles assigned to the user.
const internalRolesAttributeKey = "internalRoles"

// NewInternalRoleBasedMapper returns an implementation of RoleMapper that
// resolves the roles serialized in the "internalRoles" attribute of the user
// descriptor.
func NewInternalRoleBasedMapper() permissions.RoleMapper {
	return &internalRoleBasedMapper{}
}

type internalRoleBasedMapper struct{}

// FromUserDescriptor unmarshals the JSON payloads given by the "internalRoles"
// attribute. If the attribute is not set, no roles are resolved.
func (rm *internalRoleBasedMapper) FromUserDescriptor(_ context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	serializedRoles := user.Attributes[internalRolesAttributeKey]
	resolvedRoles := make([]permissions.ResolvedRole, 0, len(serializedRoles))
	for _, serializedRole := range serializedRoles {
		role := &tokens.InternalRole{}
		if err := role.UnmarshalJSON([]byte(serializedRole)); err != nil {
			return nil, errors.Wrap(err, "unmarshaling internal role")
		}
		resolvedRoles = append(resolvedRoles, role)
	}
	return resolvedRoles, nil
}
