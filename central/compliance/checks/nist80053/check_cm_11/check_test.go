package checkcm11

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/checks/testutils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDefaultPoliciesListUpToDate(t *testing.T) {
	defaultPolicies, err := policies.DefaultPolicies()
	require.NoError(t, err)

	for _, checkedPolicy := range defaultRuntimePackageManagementPolicies {
		t.Run(checkedPolicy, func(t *testing.T) {
			matchingIdx := sliceutils.FindMatching(defaultPolicies, func(p *storage.Policy) bool {
				return p.GetId() == checkedPolicy
			})
			require.NotEqual(t, -1, matchingIdx, "policy %q is no longer in the default policies list", checkedPolicy)
			assert.False(t, defaultPolicies[matchingIdx].GetDisabled(), "policy %q is now disabled by default", checkedPolicy)
		})
	}
}

func TestCheckAllDefaultRuntimePackageManagementPoliciesEnabled(t *testing.T) {
	for _, testCase := range []struct {
		desc       string
		policies   []testutils.LightPolicy
		shouldPass bool
	}{
		{
			desc: "all present and enabled",
			policies: []testutils.LightPolicy{
				{ID: defaultRuntimePackageManagementPolicies[0]},
				{ID: defaultRuntimePackageManagementPolicies[1]},
				{ID: defaultRuntimePackageManagementPolicies[2]},
				{ID: "Random_other"},
			},
			shouldPass: true,
		},
		{
			desc: "all present, but one not enabled",
			policies: []testutils.LightPolicy{
				{ID: defaultRuntimePackageManagementPolicies[0], Disabled: true},
				{ID: defaultRuntimePackageManagementPolicies[1]},
				{ID: defaultRuntimePackageManagementPolicies[2]},
				{ID: "Random_other"},
			},
			shouldPass: false,
		},
		{
			desc: "one missing",
			policies: []testutils.LightPolicy{
				{Name: "Ubuntu Package Manager Execution"},
				{ID: defaultRuntimePackageManagementPolicies[1]},
				{ID: defaultRuntimePackageManagementPolicies[2]},
				{ID: "Random_other"},
			},
			shouldPass: false,
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)
			testutils.MockOutLightPolicies(mockData, c.policies)
			checkAllDefaultRuntimePackageManagementPoliciesEnabled(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}
}

func TestCheckAtLeastOnePolicyTargetsAnImageRegistry(t *testing.T) {
	for _, testCase := range []struct {
		desc       string
		policies   []testutils.LightPolicy
		shouldPass bool
	}{
		{
			desc: "one policy",
			policies: []testutils.LightPolicy{
				{Name: "Ubuntu Package Manager Execution", ImageRegistry: "docker.io/stackrox"},
			},
			shouldPass: true,
		},
		{
			desc: "no such policy",
			policies: []testutils.LightPolicy{
				{Name: "Ubuntu Package Manager Execution"},
			},
			shouldPass: false,
		},
		{
			desc: "one policy, but disabled",
			policies: []testutils.LightPolicy{
				{Name: "Ubuntu Package Manager Execution", ImageRegistry: "docker.io/stackrox", Disabled: true},
			},
			shouldPass: false,
		},
	} {
		c := testCase
		t.Run(c.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)
			testutils.MockOutLightPolicies(mockData, c.policies)
			checkAtLeastOnePolicyTargetsAnImageRegistry(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}
}
