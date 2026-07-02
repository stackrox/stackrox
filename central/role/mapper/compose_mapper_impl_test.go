package mapper

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/testutils/roletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRoleMapper is a test implementation of RoleMapper
type mockRoleMapper struct {
	roles []permissions.ResolvedRole
	err   error
}

func (m *mockRoleMapper) FromUserDescriptor(_ context.Context, _ *permissions.UserDescriptor) ([]permissions.ResolvedRole, error) {
	return m.roles, m.err
}

func TestComposeMapper_FromUserDescriptor(t *testing.T) {
	tests := map[string]struct {
		mappers           []permissions.RoleMapper
		expectedRoleCount int
		expectedError     bool
		validateRoles     func(t *testing.T, roles []permissions.ResolvedRole)
	}{
		"empty mappers list": {
			mappers:           []permissions.RoleMapper{},
			expectedRoleCount: 0,
			expectedError:     false,
		},
		"single mapper with no roles": {
			mappers: []permissions.RoleMapper{
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{},
					err:   nil,
				},
			},
			expectedRoleCount: 0,
			expectedError:     false,
		},
		"single mapper with one role": {
			mappers: []permissions.RoleMapper{
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{
						roletest.NewResolvedRole(
							"admin",
							map[string]storage.Access{
								string(resources.Namespace.GetResource()): storage.Access_READ_WRITE_ACCESS,
							},
							nil,
						),
					},
					err: nil,
				},
			},
			expectedRoleCount: 1,
			expectedError:     false,
			validateRoles: func(t *testing.T, roles []permissions.ResolvedRole) {
				require.Len(t, roles, 1)
				assert.Equal(t, "admin", roles[0].GetRoleName())
			},
		},
		"multiple mappers each with one role": {
			mappers: []permissions.RoleMapper{
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{
						roletest.NewResolvedRole(
							"viewer",
							map[string]storage.Access{
								string(resources.Namespace.GetResource()): storage.Access_READ_ACCESS,
							},
							nil,
						),
					},
					err: nil,
				},
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{
						roletest.NewResolvedRole(
							"editor",
							map[string]storage.Access{
								string(resources.Secret.GetResource()): storage.Access_READ_WRITE_ACCESS,
							},
							nil,
						),
					},
					err: nil,
				},
			},
			expectedRoleCount: 2,
			expectedError:     false,
			validateRoles: func(t *testing.T, roles []permissions.ResolvedRole) {
				require.Len(t, roles, 2)
				assert.Equal(t, "viewer", roles[0].GetRoleName())
				assert.Equal(t, "editor", roles[1].GetRoleName())
			},
		},
		"multiple mappers with multiple roles each": {
			mappers: []permissions.RoleMapper{
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{
						roletest.NewResolvedRole("role1", map[string]storage.Access{}, nil),
						roletest.NewResolvedRole("role2", map[string]storage.Access{}, nil),
					},
					err: nil,
				},
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{
						roletest.NewResolvedRole("role3", map[string]storage.Access{}, nil),
					},
					err: nil,
				},
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{
						roletest.NewResolvedRole("role4", map[string]storage.Access{}, nil),
						roletest.NewResolvedRole("role5", map[string]storage.Access{}, nil),
					},
					err: nil,
				},
			},
			expectedRoleCount: 5,
			expectedError:     false,
			validateRoles: func(t *testing.T, roles []permissions.ResolvedRole) {
				require.Len(t, roles, 5)
				assert.Equal(t, "role1", roles[0].GetRoleName())
				assert.Equal(t, "role2", roles[1].GetRoleName())
				assert.Equal(t, "role3", roles[2].GetRoleName())
				assert.Equal(t, "role4", roles[3].GetRoleName())
				assert.Equal(t, "role5", roles[4].GetRoleName())
			},
		},
		"error from first mapper": {
			mappers: []permissions.RoleMapper{
				&mockRoleMapper{
					roles: nil,
					err:   errors.New("mapper 1 failed"),
				},
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{
						roletest.NewResolvedRole("role1", map[string]storage.Access{}, nil),
					},
					err: nil,
				},
			},
			expectedRoleCount: 0,
			expectedError:     true,
		},
		"error from second mapper": {
			mappers: []permissions.RoleMapper{
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{
						roletest.NewResolvedRole("role1", map[string]storage.Access{}, nil),
					},
					err: nil,
				},
				&mockRoleMapper{
					roles: nil,
					err:   errors.New("mapper 2 failed"),
				},
			},
			expectedRoleCount: 0,
			expectedError:     true,
		},
		"mixed empty and non-empty mappers": {
			mappers: []permissions.RoleMapper{
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{},
					err:   nil,
				},
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{
						roletest.NewResolvedRole("role1", map[string]storage.Access{}, nil),
					},
					err: nil,
				},
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{},
					err:   nil,
				},
				&mockRoleMapper{
					roles: []permissions.ResolvedRole{
						roletest.NewResolvedRole("role2", map[string]storage.Access{}, nil),
					},
					err: nil,
				},
			},
			expectedRoleCount: 2,
			expectedError:     false,
			validateRoles: func(t *testing.T, roles []permissions.ResolvedRole) {
				require.Len(t, roles, 2)
				assert.Equal(t, "role1", roles[0].GetRoleName())
				assert.Equal(t, "role2", roles[1].GetRoleName())
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			composeMapper := NewComposeMapper(tc.mappers...)

			userDescriptor := &permissions.UserDescriptor{
				UserID:     "test-user",
				Attributes: map[string][]string{},
			}

			roles, err := composeMapper.FromUserDescriptor(context.Background(), userDescriptor)

			if tc.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, roles, tc.expectedRoleCount)

			if tc.validateRoles != nil {
				tc.validateRoles(t, roles)
			}
		})
	}
}

func TestNewComposeMapper(t *testing.T) {
	mapper1 := &mockRoleMapper{
		roles: []permissions.ResolvedRole{
			roletest.NewResolvedRole("role1", map[string]storage.Access{}, nil),
		},
		err: nil,
	}
	mapper2 := &mockRoleMapper{
		roles: []permissions.ResolvedRole{
			roletest.NewResolvedRole("role2", map[string]storage.Access{}, nil),
		},
		err: nil,
	}

	composeMapper := NewComposeMapper(mapper1, mapper2)
	require.NotNil(t, composeMapper)

	// Verify it implements RoleMapper interface
	var _ permissions.RoleMapper = composeMapper
}

func TestComposeMapper_PreservesOrder(t *testing.T) {
	// Verify that roles are returned in the order of mappers
	mappers := []permissions.RoleMapper{
		&mockRoleMapper{
			roles: []permissions.ResolvedRole{
				roletest.NewResolvedRole("first-1", map[string]storage.Access{}, nil),
				roletest.NewResolvedRole("first-2", map[string]storage.Access{}, nil),
			},
			err: nil,
		},
		&mockRoleMapper{
			roles: []permissions.ResolvedRole{
				roletest.NewResolvedRole("second-1", map[string]storage.Access{}, nil),
			},
			err: nil,
		},
		&mockRoleMapper{
			roles: []permissions.ResolvedRole{
				roletest.NewResolvedRole("third-1", map[string]storage.Access{}, nil),
				roletest.NewResolvedRole("third-2", map[string]storage.Access{}, nil),
				roletest.NewResolvedRole("third-3", map[string]storage.Access{}, nil),
			},
			err: nil,
		},
	}

	composeMapper := NewComposeMapper(mappers...)
	userDescriptor := &permissions.UserDescriptor{
		UserID:     "test-user",
		Attributes: map[string][]string{},
	}

	roles, err := composeMapper.FromUserDescriptor(context.Background(), userDescriptor)

	require.NoError(t, err)
	require.Len(t, roles, 6)

	expectedOrder := []string{"first-1", "first-2", "second-1", "third-1", "third-2", "third-3"}
	for i, expectedName := range expectedOrder {
		assert.Equal(t, expectedName, roles[i].GetRoleName(),
			"Role at position %d should be %s", i, expectedName)
	}
}
