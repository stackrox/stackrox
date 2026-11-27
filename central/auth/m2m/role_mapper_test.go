package m2m

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

type testResolvedRole struct {
	permissions.ResolvedRole
	name string
}

func (t *testResolvedRole) GetRoleName() string {
	return t.name
}

func TestResolveRolesForClaims(t *testing.T) {
	claims := map[string][]string{
		"sub":        {"something"},
		"aud":        {"something", "somewhere"},
		"repository": {"github.com/sample-org/sample-repo:main:062348SHA"},
		"iss":        {"https://stackrox.io"},
		"groups":     {"system:admins"},
	}
	config := &storage.AuthMachineToMachineConfig{
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:             "sub",
				ValueExpression: "something",
				Role:            "Admin",
			},
			{
				Key:             "aud",
				ValueExpression: "somewhere",
				Role:            "Analyst",
			},
			{
				Key:             "aud",
				ValueExpression: "something",
				Role:            "Analyst",
			},
			{
				Key:             "aud",
				ValueExpression: "elsewhere",
				Role:            "Continuous Integration",
			},
			{
				Key:             "repository",
				ValueExpression: "github.com/sample-org/sample-repo.*",
				Role:            "roxctl",
			},
			{
				Key:             "iss",
				ValueExpression: ".*",
				Role:            authn.NoneRole,
			}, {
				Key:             "groups",
				ValueExpression: "system:admins",
				Role:            "Analyst",
			},
		},
	}
	roles := map[string]permissions.ResolvedRole{
		"Admin":        &testResolvedRole{name: "Admin"},
		"Analyst":      &testResolvedRole{name: "Analyst"},
		"roxctl":       &testResolvedRole{name: "roxctl"},
		authn.NoneRole: &testResolvedRole{name: authn.NoneRole},
	}

	roleDSMock := mocks.NewMockDataStore(gomock.NewController(t))

	for roleName, resolvedRole := range roles {
		roleDSMock.EXPECT().GetAndResolveRole(gomock.Any(), roleName).Return(resolvedRole, nil)
	}
	configMappings := parseMappings(config.GetMappings())

	resolvedRoles, err := resolveRolesForClaims(context.Background(), claims, roleDSMock, configMappings)
	assert.NoError(t, err)
	assert.ElementsMatch(t, resolvedRoles, []permissions.ResolvedRole{roles["Admin"], roles["Analyst"], roles["roxctl"]})
}

func TestRoleMapperFromUserDescriptor(t *testing.T) {
	testCases := map[string]struct {
		claims        map[string][]string
		mappings      []*storage.AuthMachineToMachineConfig_Mapping
		roles         map[string]permissions.ResolvedRole
		roleLookupErr error
		expectedRoles []permissions.ResolvedRole
		expectedError error
	}{
		"maps matching claims to roles": {
			claims: map[string][]string{
				"sub":        {"something"},
				"aud":        {"something", "somewhere"},
				"repository": {"github.com/sample-org/sample-repo:main:062348SHA"},
				"groups":     {"system:admins"},
			},
			mappings: []*storage.AuthMachineToMachineConfig_Mapping{
				{
					Key:             "sub",
					ValueExpression: "something",
					Role:            "Admin",
				},
				{
					Key:             "aud",
					ValueExpression: "somewhere",
					Role:            "Analyst",
				},
				{
					Key:             "aud",
					ValueExpression: "elsewhere",
					Role:            "Continuous Integration",
				},
				{
					Key:             "repository",
					ValueExpression: "github.com/sample-org/sample-repo.*",
					Role:            "roxctl",
				},
				{
					Key:             "groups",
					ValueExpression: "system:admins",
					Role:            "Analyst",
				},
			},
			roles: map[string]permissions.ResolvedRole{
				"Admin":   &testResolvedRole{name: "Admin"},
				"Analyst": &testResolvedRole{name: "Analyst"},
				"roxctl":  &testResolvedRole{name: "roxctl"},
			},
			expectedRoles: []permissions.ResolvedRole{
				&testResolvedRole{name: "Admin"},
				&testResolvedRole{name: "Analyst"},
				&testResolvedRole{name: "roxctl"},
			},
		},
		"skips the none role": {
			claims: map[string][]string{
				"iss": {"https://stackrox.io"},
			},
			mappings: []*storage.AuthMachineToMachineConfig_Mapping{
				{
					Key:             "iss",
					ValueExpression: ".*",
					Role:            authn.NoneRole,
				},
			},
			roles: map[string]permissions.ResolvedRole{
				authn.NoneRole: &testResolvedRole{name: authn.NoneRole},
			},
			expectedRoles: nil,
		},
		"returns error if no mapping matches": {
			claims: map[string][]string{
				"sub": {"something"},
			},
			mappings: []*storage.AuthMachineToMachineConfig_Mapping{
				{
					Key:             "sub",
					ValueExpression: "something-else",
					Role:            "Admin",
				},
			},
			expectedError: auth.ErrNoValidRole,
		},
		"returns error if role resolution fails": {
			claims: map[string][]string{
				"sub": {"something"},
			},
			mappings: []*storage.AuthMachineToMachineConfig_Mapping{
				{
					Key:             "sub",
					ValueExpression: "something",
					Role:            "Admin",
				},
			},
			roles: map[string]permissions.ResolvedRole{
				"Admin": &testResolvedRole{name: "Admin"},
			},
			roleLookupErr: assert.AnError,
			expectedError: assert.AnError,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(it *testing.T) {
			roleDSMock := mocks.NewMockDataStore(gomock.NewController(it))
			for roleName, resolvedRole := range tc.roles {
				roleDSMock.EXPECT().GetAndResolveRole(gomock.Any(), roleName).Return(resolvedRole, tc.roleLookupErr).AnyTimes()
			}

			mapper := newRoleMapperFromMappings(tc.mappings, roleDSMock)
			user := &permissions.UserDescriptor{
				UserID:     "test-user",
				Attributes: tc.claims,
			}

			resolvedRoles, err := mapper.FromUserDescriptor(it.Context(), user)
			if tc.expectedError != nil {
				assert.ErrorIs(it, err, tc.expectedError)
				return
			}
			assert.NoError(it, err)
			assert.ElementsMatch(it, resolvedRoles, tc.expectedRoles)
		})
	}
}

func TestResolveRolesForClaimsList(t *testing.T) {
	claims := map[string][]string{
		"groups": {"system:admins"},
	}
	config := &storage.AuthMachineToMachineConfig{
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			{
				Key:             "groups",
				ValueExpression: "system:admins",
				Role:            "Analyst",
			},
		},
	}
	roles := map[string]permissions.ResolvedRole{
		"Analyst": &testResolvedRole{name: "Analyst"},
	}

	roleDSMock := mocks.NewMockDataStore(gomock.NewController(t))

	for roleName, resolvedRole := range roles {
		roleDSMock.EXPECT().GetAndResolveRole(gomock.Any(), roleName).Return(resolvedRole, nil)
	}
	configMappings := parseMappings(config.GetMappings())

	resolvedRoles, err := resolveRolesForClaims(t.Context(), claims, roleDSMock, configMappings)
	assert.NoError(t, err)
	assert.ElementsMatch(t, resolvedRoles, []permissions.ResolvedRole{roles["Analyst"]})
}
