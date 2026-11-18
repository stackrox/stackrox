package openshift

import (
	"context"

	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/auth/permissions"
	permissionsUtils "github.com/stackrox/rox/pkg/auth/permissions/utils"
	"github.com/stackrox/rox/pkg/errox"
)

type metricsReaderRoleMapperFactory struct {
	delegate permissions.RoleMapperFactory
	roles    roleDataStore.DataStore
}

// WrapMetricsReaderRoleMapperFactory validates the direct OpenShift reader's permission set at role resolution.
func WrapMetricsReaderRoleMapperFactory(delegate permissions.RoleMapperFactory, roles roleDataStore.DataStore) permissions.RoleMapperFactory {
	return &metricsReaderRoleMapperFactory{delegate: delegate, roles: roles}
}

func (f *metricsReaderRoleMapperFactory) GetRoleMapper(providerID string) permissions.RoleMapper {
	delegate := f.delegate.GetRoleMapper(providerID)
	if providerID != authProviderID {
		return delegate
	}
	return &metricsReaderRoleMapper{delegate: delegate, roles: f.roles}
}

type metricsReaderRoleMapper struct {
	delegate permissions.RoleMapper
	roles    roleDataStore.DataStore
}

func (m *metricsReaderRoleMapper) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	resolved, err := m.delegate.FromUserDescriptor(ctx, user)
	if err != nil {
		return nil, err
	}
	for _, role := range resolved {
		if role == nil || role.GetRoleName() != roleName {
			continue
		}
		stored, found, err := m.roles.GetRole(ctx, roleName)
		if err != nil {
			return nil, err
		}
		if !found || stored == nil {
			return nil, errox.NotAuthorized.New("OpenShift metrics reader role is unavailable")
		}
		set, found, err := m.roles.GetPermissionSet(ctx, stored.GetPermissionSetId())
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errox.NotAuthorized.New("OpenShift metrics reader permission set is unavailable")
		}
		if err := permissionsUtils.ValidateScopedReadOnlyPermissionSet(set); err != nil {
			return nil, errox.NotAuthorized.CausedByf("OpenShift metrics reader permission set: %v", err)
		}
	}
	return resolved, nil
}
