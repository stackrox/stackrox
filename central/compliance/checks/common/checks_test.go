package common

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/compliance/checks/testutils"
	"github.com/stackrox/rox/pkg/features"
	pkgTestUtils "github.com/stackrox/rox/pkg/testutils"
)

func TestCheckSecretsInEnv(t *testing.T) {
	for _, testCase := range []struct {
		desc        string
		policies    []testutils.LightPolicy
		expectedIDs []string
		shouldPass  bool
	}{
		{
			desc: "no policies with secrets in env",
			policies: []testutils.LightPolicy{
				{Name: "Definitely not about secrets"},
				{Name: "Random other"},
			},
			shouldPass: false,
		},
		{
			desc: "one policy with secrets in env, not enforced",
			policies: []testutils.LightPolicy{
				{Name: "Definitely about secrets", EnvKey: "this_is_secret", EnvValue: "DONTLOOKATME"},
				{Name: "Random other"},
			},
			shouldPass: false,
		},
		{
			desc: "one policy with secrets in env, enforced",
			policies: []testutils.LightPolicy{
				{Name: "Definitely about secrets", EnvKey: "this_is_secret", EnvValue: "DONTLOOKATME", Enforced: true},
				{Name: "Random other"},
			},
			shouldPass: true,
		},
		{
			desc: "one policy with secrets in env, enforced but disabled",
			policies: []testutils.LightPolicy{
				{Name: "Definitely about secrets", EnvKey: "this_is_secret", EnvValue: "DONTLOOKATME", Disabled: true, Enforced: true},
				{Name: "Random other"},
			},
			shouldPass: false,
		},
		{
			desc: "one policy with secrets in env, enforced but no value",
			policies: []testutils.LightPolicy{
				{Name: "Definitely about secrets", EnvKey: "this_is_secret", Enforced: true},
				{Name: "Random other"},
			},
			shouldPass: false,
		},
	} {
		c := testCase
		pkgTestUtils.RunWithAndWithoutFeatureFlag(t, features.BooleanPolicyLogic.EnvVar(), c.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)
			testutils.MockOutLightPolicies(mockData, c.policies)
			CheckSecretsInEnv(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}
}
