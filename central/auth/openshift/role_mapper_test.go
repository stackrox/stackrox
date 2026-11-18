package openshift

import (
	"context"
	"testing"

	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testRoleMapper func(context.Context, *permissions.UserDescriptor) ([]permissions.ResolvedRole, error)

func (m testRoleMapper) FromUserDescriptor(ctx context.Context, user *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	return m(ctx, user)
}

type testRoleMapperFactory func(string) permissions.RoleMapper

func (f testRoleMapperFactory) GetRoleMapper(providerID string) permissions.RoleMapper {
	return f(providerID)
}

func TestMetricsReaderRoleMapper(t *testing.T) {
	for name, tc := range map[string]struct {
		set     *storage.PermissionSet
		wantErr bool
	}{
		"constrained": {set: &storage.PermissionSet{
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
			ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS},
		}},
		"unconstrained": {set: &storage.PermissionSet{
			ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS},
		}, wantErr: true},
		"invalid stored grant": {set: &storage.PermissionSet{
			AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
			ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
			ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_WRITE_ACCESS},
		}, wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			roles := roleMocks.NewMockDataStore(gomock.NewController(t))
			tc.set.Id = "8df4a4a0-a292-4e33-a076-fab229cbbcea"
			roles.EXPECT().GetRole(gomock.Any(), roleName).Return(
				&storage.Role{Name: roleName, PermissionSetId: tc.set.GetId()}, true, nil)
			roles.EXPECT().GetPermissionSet(gomock.Any(), tc.set.GetId()).Return(tc.set, true, nil)
			delegate := testRoleMapper(func(context.Context, *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
				rolePermissions := make(map[storage.Access][]string)
				for resource, access := range tc.set.GetResourceToAccess() {
					rolePermissions[access] = append(rolePermissions[access], resource)
				}
				return []permissions.ResolvedRole{&tokens.InternalRole{
					RoleName: roleName, Permissions: rolePermissions,
				}}, nil
			})
			factory := WrapMetricsReaderRoleMapperFactory(
				testRoleMapperFactory(func(string) permissions.RoleMapper { return delegate }), roles)
			resolved, err := factory.GetRoleMapper(authProviderID).FromUserDescriptor(t.Context(), &permissions.UserDescriptor{})
			if tc.wantErr {
				require.ErrorIs(t, err, errox.NotAuthorized)
				assert.Nil(t, resolved)
			} else {
				require.NoError(t, err)
				require.Len(t, resolved, 1)
			}
			other := factory.GetRoleMapper("other")
			_, err = other.FromUserDescriptor(t.Context(), &permissions.UserDescriptor{})
			require.NoError(t, err)
		})
	}
}

func TestMetricsReaderRoleMapperChecksCurrentConstraints(t *testing.T) {
	roles := roleMocks.NewMockDataStore(gomock.NewController(t))
	set := &storage.PermissionSet{
		Id:               "8df4a4a0-a292-4e33-a076-fab229cbbcea",
		AccessConstraint: storage.AccessConstraint_ACCESS_CONSTRAINT_READ,
		ScopeConstraint:  storage.ScopeConstraint_SCOPE_CONSTRAINT_CLUSTER_NAMESPACE,
		ResourceToAccess: map[string]storage.Access{"Deployment": storage.Access_READ_ACCESS},
	}
	roles.EXPECT().GetRole(gomock.Any(), roleName).Return(
		&storage.Role{Name: roleName, PermissionSetId: set.GetId()}, true, nil).Times(2)
	roles.EXPECT().GetPermissionSet(gomock.Any(), set.GetId()).Return(set, true, nil).Times(2)
	delegate := testRoleMapper(func(context.Context, *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
		return []permissions.ResolvedRole{&tokens.InternalRole{
			RoleName:    roleName,
			Permissions: map[storage.Access][]string{storage.Access_READ_ACCESS: {"Deployment"}},
		}}, nil
	})
	factory := WrapMetricsReaderRoleMapperFactory(
		testRoleMapperFactory(func(string) permissions.RoleMapper { return delegate }), roles)
	mapper := factory.GetRoleMapper(authProviderID)

	_, err := mapper.FromUserDescriptor(t.Context(), &permissions.UserDescriptor{})
	require.NoError(t, err)
	set.AccessConstraint = storage.AccessConstraint_ACCESS_CONSTRAINT_UNSPECIFIED
	_, err = mapper.FromUserDescriptor(t.Context(), &permissions.UserDescriptor{})
	require.ErrorIs(t, err, errox.NotAuthorized)
}
