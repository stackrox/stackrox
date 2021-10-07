package check

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestPolicyResourceTransformer_JSONFormat(t *testing.T) {
	cases := map[string]struct {
		alerts         []*storage.Alert
		failedPolicies []*storage.Policy
		summary        map[string]int
		expectedOutput *policyJSONResult
	}{
		"empty alerts and failed policies": {
			alerts:         nil,
			summary:        map[string]int{"TOTAL": 0, "LOW": 0, "MEDIUM": 0, "HIGH": 0, "CRITICAL": 0},
			failedPolicies: nil,
			expectedOutput: &policyJSONResult{
				policyJSONStructure{
					Summary: map[string]int{
						"TOTAL":    0,
						"LOW":      0,
						"MEDIUM":   0,
						"HIGH":     0,
						"CRITICAL": 0,
					},
					ViolatedPolicies:      []policyJSON{},
					BuildBreakingPolicies: []breakingPolicyJSON{},
				},
			},
		},
		"policy violations but no failed policies": {
			alerts: []*storage.Alert{
				{
					Policy: &storage.Policy{
						Id:          "policy1",
						Name:        "test policy 1",
						Description: "test description 1",
						Remediation: "test remediation 1",
						Severity:    storage.Severity_HIGH_SEVERITY,
					},
					Violations: []*storage.Alert_Violation{
						{
							Message: "test violation 1",
						},
						{
							Message: "test violation 2",
						},
					},
				},
			},
			summary:        map[string]int{"HIGH": 1, "TOTAL": 1, "LOW": 0, "MEDIUM": 0, "CRITICAL": 0},
			failedPolicies: nil,
			expectedOutput: &policyJSONResult{
				policyJSONStructure{
					Summary: map[string]int{
						"TOTAL":    1,
						"LOW":      0,
						"MEDIUM":   0,
						"HIGH":     1,
						"CRITICAL": 0,
					},
					ViolatedPolicies: []policyJSON{
						{
							Name:        "test policy 1",
							Severity:    "HIGH",
							Description: "test description 1",
							Violation:   "- test violation 1\n- test violation 2\n",
							Remediation: "test remediation 1",
						},
					},
					BuildBreakingPolicies: []breakingPolicyJSON{},
				},
			},
		},
		"policy violations with failed policies": {
			alerts: []*storage.Alert{
				{
					Policy: &storage.Policy{
						Id:          "policy1",
						Name:        "test policy 1",
						Description: "test description 1",
						Remediation: "test remediation 1",
						Severity:    storage.Severity_HIGH_SEVERITY,
					},
					Violations: []*storage.Alert_Violation{
						{
							Message: "test violation 1",
						},
						{
							Message: "test violation 2",
						},
					},
				},
				{
					Policy: &storage.Policy{
						Id:          "policy2",
						Name:        "test policy 2",
						Description: "test description 2",
						Remediation: "test remediation 2",
						Severity:    storage.Severity_LOW_SEVERITY,
					},
					Violations: []*storage.Alert_Violation{
						{
							Message: "test violation 1",
						},
						{
							Message: "test violation 2",
						},
					},
				},
			},
			summary: map[string]int{"TOTAL": 2, "HIGH": 1, "LOW": 1, "MEDIUM": 0, "CRITICAL": 0},
			failedPolicies: []*storage.Policy{
				{
					Name:        "test policy 2",
					Remediation: "test remediation 2",
				},
			},
			expectedOutput: &policyJSONResult{
				policyJSONStructure{
					Summary: map[string]int{
						"TOTAL":    2,
						"LOW":      1,
						"MEDIUM":   0,
						"HIGH":     1,
						"CRITICAL": 0,
					},
					BuildBreakingPolicies: []breakingPolicyJSON{
						{
							Name:        "test policy 2",
							Remediation: "test remediation 2",
						},
					},
					ViolatedPolicies: []policyJSON{
						{
							Name:        "test policy 1",
							Severity:    "HIGH",
							Description: "test description 1",
							Violation:   "- test violation 1\n- test violation 2\n",
							Remediation: "test remediation 1",
						},
						{
							Name:        "test policy 2",
							Severity:    "LOW",
							Description: "test description 2",
							Violation:   "- test violation 1\n- test violation 2\n",
							Remediation: "test remediation 2",
						},
					},
				},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			alertSummary := newAlertSummaryForPrinting(c.alerts, c.failedPolicies, c.summary)
			assert.Equal(t, c.expectedOutput, alertSummary.(*policyJSONResult))
		})
	}
}
