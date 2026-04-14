package mapper

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

type composeMapperImpl struct {
	mappers []permissions.RoleMapper
}

// FromUserDescriptor calls FromUserDescriptor on all internal mappers and concatenates the results.
// If any mapper returns an error, the error is returned immediately.
func (cm *composeMapperImpl) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	var allRoles []permissions.ResolvedRole

	for _, mapper := range cm.mappers {
		roles, err := mapper.FromUserDescriptor(ctx, user)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get roles from mapper")
		}
		allRoles = append(allRoles, roles...)
	}

	return allRoles, nil
}

// NewComposeMapper creates a RoleMapper that delegates to multiple mappers and concatenates their results.
func NewComposeMapper(mappers ...permissions.RoleMapper) permissions.RoleMapper {
	return &composeMapperImpl{
		mappers: mappers,
	}
}
