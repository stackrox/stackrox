package common

import (
	"testing"

	"github.com/stackrox/rox/central/compliance/checks/testutils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCheckAtLeastOnePolicyEnabledReferringToVulns(t *testing.T) {
	for _, testCase := range []struct {
		desc        string
		policies    []testutils.LightPolicy
		expectedIDs []string
		shouldPass  bool
	}{
		{
			desc: "no policies referring to vulns",
			policies: []testutils.LightPolicy{
				{Name: "Definitely not about vulns"},
				{Name: "Random other"},
			},
			shouldPass: false,
		},
		{
			desc: "one CVSS policy",
			policies: []testutils.LightPolicy{
				{ID: "blah", Name: "Bad CVSS is bad", CVSSGreaterThan: 6},
			},
			expectedIDs: []string{"blah"},
			shouldPass:  true,
		},
		{
			desc: "one CVSS policy, disabled",
			policies: []testutils.LightPolicy{
				{Name: "Bad CVSS is bad", CVSSGreaterThan: 6, Disabled: true},
			},
			shouldPass: false,
		},
		{
			desc: "one CVE policy",
			policies: []testutils.LightPolicy{
				{ID: "anycve", Name: "Any CVE", CVE: ".*"},
			},
			expectedIDs: []string{"anycve"},
			shouldPass:  true,
		},
		{
			desc: "another CVE policy",
			policies: []testutils.LightPolicy{
				{ID: "anycve", Name: "Any CVE", CVE: "CVE-2017-.+"},
			},
			expectedIDs: []string{"anycve"},
			shouldPass:  true,
		},
		{
			desc: "exact CVE policy",
			policies: []testutils.LightPolicy{
				{Name: "Any CVE", CVE: "CVE-2017-1234"},
			},
			shouldPass: false,
		},
		{
			desc: "one CVE policy, disabled",
			policies: []testutils.LightPolicy{
				{Name: "Any CVE", CVE: ".*", Disabled: true},
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
			policyIDSet := CheckAtLeastOnePolicyEnabledReferringToVulns(mockCtx)
			records.AssertExpectedResult(c.shouldPass, t)
			assert.ElementsMatch(t, c.expectedIDs, policyIDSet.AsSlice())
		})
	}
}
