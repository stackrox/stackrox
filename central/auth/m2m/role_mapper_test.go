package m2m

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
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
	}
	config := storage.AuthMachineToMachineConfig_builder{
		Mappings: []*storage.AuthMachineToMachineConfig_Mapping{
			storage.AuthMachineToMachineConfig_Mapping_builder{
				Key:             "sub",
				ValueExpression: "something",
				Role:            "Admin",
			}.Build(),
			storage.AuthMachineToMachineConfig_Mapping_builder{
				Key:             "aud",
				ValueExpression: "somewhere",
				Role:            "Analyst",
			}.Build(),
			storage.AuthMachineToMachineConfig_Mapping_builder{
				Key:             "aud",
				ValueExpression: "something",
				Role:            "Analyst",
			}.Build(),
			storage.AuthMachineToMachineConfig_Mapping_builder{
				Key:             "aud",
				ValueExpression: "elsewhere",
				Role:            "Continuous Integration",
			}.Build(),
			storage.AuthMachineToMachineConfig_Mapping_builder{
				Key:             "repository",
				ValueExpression: "github.com/sample-org/sample-repo.*",
				Role:            "roxctl",
			}.Build(),
			storage.AuthMachineToMachineConfig_Mapping_builder{
				Key:             "iss",
				ValueExpression: ".*",
				Role:            authn.NoneRole,
			}.Build(),
		},
	}.Build()
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

	resolvedRoles, err := resolveRolesForClaims(context.Background(), claims, roleDSMock, config.GetMappings(), createRegexp(config))
	assert.NoError(t, err)
	assert.ElementsMatch(t, resolvedRoles, []permissions.ResolvedRole{roles["Admin"], roles["Analyst"], roles["roxctl"]})
}
