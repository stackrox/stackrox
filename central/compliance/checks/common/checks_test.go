package common

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/checks/testutils"
	"github.com/stackrox/rox/generated/storage"
	"go.uber.org/mock/gomock"
)

func TestCheckSecretsInEnv(t *testing.T) {
	for desc, testCase := range map[string]struct {
		policies   []testutils.LightPolicy
		shouldPass bool
	}{
		"no policies with secrets in env": {
			policies: []testutils.LightPolicy{
				{Name: "Definitely not about secrets"},
				{Name: "Random other"},
			},
			shouldPass: false,
		},
		"one policy with secrets in env, not enforced": {
			policies: []testutils.LightPolicy{
				{Name: "Definitely about secrets", EnvKey: "this_is_secret", EnvValue: "DONTLOOKATME"},
				{Name: "Random other"},
			},
			shouldPass: false,
		},
		"one policy with secrets in env, enforced": {
			policies: []testutils.LightPolicy{
				{Name: "Definitely about secrets", EnvKey: "this_is_secret", EnvValue: "DONTLOOKATME", Enforced: true},
				{Name: "Random other"},
			},
			shouldPass: true,
		},
		"another policy with secrets in env, enforced": {
			policies: []testutils.LightPolicy{
				{Name: "Definitely about secrets", EnvKey: ".*SECRET.*|.*PASSWORD.*", EnvValue: "", Enforced: true},
				{Name: "Random other"},
			},
			shouldPass: true,
		},

		"one policy with secrets in env, enforced but disabled": {
			policies: []testutils.LightPolicy{
				{Name: "Definitely about secrets", EnvKey: "this_is_secret", EnvValue: "DONTLOOKATME", Disabled: true, Enforced: true},
				{Name: "Random other"},
			},
			shouldPass: false,
		},
	} {
		c := testCase
		t.Run(desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)
			testutils.MockOutLightPolicies(mockData, c.policies)
			CheckSecretsInEnv(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}
}

func TestCheckNotifierInUseByCluster(t *testing.T) {
	var (
		someNotifier = storage.Notifier{
			Id: "Some Notifier",
		}
		someOtherNotifier = storage.Notifier{
			Id: "Some Other Notifier",
		}
		notifiers = []*storage.Notifier{&someNotifier, &someOtherNotifier}
	)

	for desc, testCase := range map[string]struct {
		policies   []testutils.LightPolicy
		notifiers  []*storage.Notifier
		shouldPass bool
	}{
		"no policies": {
			policies:   []testutils.LightPolicy{},
			notifiers:  notifiers,
			shouldPass: false,
		},
		"one policy enabled but no notifier": {
			policies: []testutils.LightPolicy{
				{Name: "Some Policy"},
			},
			notifiers:  notifiers,
			shouldPass: false,
		},
		"one policy with non-matching notifier (bad ID?)": {
			policies: []testutils.LightPolicy{
				{Name: "Some Policy", Notifiers: []string{"Non-existent Notifier"}},
			},
			notifiers:  notifiers,
			shouldPass: false,
		},
		"one policy with matching notifier but not enabled": {

			policies: []testutils.LightPolicy{
				{Name: "Some Policy", Disabled: true, Notifiers: []string{someNotifier.Id}},
			},
			notifiers:  notifiers,
			shouldPass: false,
		},
		"one policy with matching notifier": {
			policies: []testutils.LightPolicy{
				{Name: "Some Policy", Notifiers: []string{someNotifier.Id}},
			},
			notifiers:  notifiers,
			shouldPass: true,
		},
		"two policies each with no notifier": {
			policies: []testutils.LightPolicy{
				{Name: "Policy one"},
				{Name: "Policy two"},
			},
			notifiers:  notifiers,
			shouldPass: false,
		},
		"two policies second one with notifier": {
			policies: []testutils.LightPolicy{
				{Name: "Policy one"},
				{Name: "Policy two", Notifiers: []string{someNotifier.Id}},
			},
			notifiers:  notifiers,
			shouldPass: true,
		},
	} {
		c := testCase
		t.Run(desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockCtx, mockData, records := testutils.SetupMockCtxAndMockData(ctrl)
			testutils.MockOutLightPolicies(mockData, c.policies)
			mockData.EXPECT().Notifiers().Return(c.notifiers)
			CheckNotifierInUseByCluster(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
		})
	}
}
