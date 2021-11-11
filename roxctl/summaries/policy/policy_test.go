package policy

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestNewPolicySummaryForPrinting(t *testing.T) {
	cases := map[string]struct {
		alerts         []*storage.Alert
		expectedOutput *Result
	}{
		"empty alerts and failed policies": {
			expectedOutput: &Result{
				Results: []EntityResult{},
				Summary: map[string]int{
					"TOTAL":    0,
					"LOW":      0,
					"MEDIUM":   0,
					"HIGH":     0,
					"CRITICAL": 0,
				},
			},
		},
		"Policy violations but no failed policies - unknown entity": {
			alerts: []*storage.Alert{
				{
					Policy: &storage.Policy{
						Id:          "policy1",
						Name:        "test Policy 1",
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
			expectedOutput: &Result{
				Results: []EntityResult{
					{
						Summary: map[string]int{
							"TOTAL":    1,
							"LOW":      0,
							"MEDIUM":   0,
							"HIGH":     1,
							"CRITICAL": 0,
						},
						ViolatedPolicies: []Policy{
							{
								Name:        "test Policy 1",
								Severity:    "HIGH",
								Description: "test description 1",
								Violation:   []string{"test violation 1", "test violation 2"},
								Remediation: "test remediation 1",
							},
						},
						Metadata: EntityMetadata{ID: "unknown"},
					},
				},
				Summary: map[string]int{
					"TOTAL":    1,
					"LOW":      0,
					"MEDIUM":   0,
					"HIGH":     1,
					"CRITICAL": 0,
				},
			},
		},
		"Policy violations with failed policies - unknown entity": {
			alerts: []*storage.Alert{
				{
					Policy: &storage.Policy{
						Id:          "policy1",
						Name:        "test Policy 1",
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
						Id:                 "policy2",
						Name:               "test Policy 2",
						Description:        "test description 2",
						Remediation:        "test remediation 2",
						Severity:           storage.Severity_LOW_SEVERITY,
						EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
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
			expectedOutput: &Result{
				Results: []EntityResult{
					{
						Summary: map[string]int{
							"TOTAL":    2,
							"LOW":      1,
							"MEDIUM":   0,
							"HIGH":     1,
							"CRITICAL": 0,
						},
						ViolatedPolicies: []Policy{
							{
								Name:        "test Policy 1",
								Severity:    "HIGH",
								Description: "test description 1",
								Violation:   []string{"test violation 1", "test violation 2"},
								Remediation: "test remediation 1",
							},
							{
								Name:         "test Policy 2",
								Severity:     "LOW",
								Description:  "test description 2",
								Violation:    []string{"test violation 1", "test violation 2"},
								Remediation:  "test remediation 2",
								FailingCheck: true,
							},
						},
						Metadata: EntityMetadata{ID: "unknown"},
					},
				},
				Summary: map[string]int{
					"TOTAL":    2,
					"LOW":      1,
					"MEDIUM":   0,
					"HIGH":     1,
					"CRITICAL": 0,
				},
			},
		},
		"multiple entities within alerts": {
			alerts: []*storage.Alert{
				{
					Entity: &storage.Alert_Image{
						Image: &storage.ContainerImage{
							Id:   "nginx",
							Name: &storage.ImageName{FullName: "nginx"},
						},
					},
					Policy: &storage.Policy{
						Id:          "policy1",
						Name:        "test Policy 1",
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
					Entity: &storage.Alert_Deployment_{
						Deployment: &storage.Alert_Deployment{
							Id:        "deployment",
							Name:      "test-deployment",
							Type:      "deployment",
							Namespace: "default",
						},
					},
					Policy: &storage.Policy{
						Id:          "policy1",
						Name:        "test Policy 1",
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
					Entity: &storage.Alert_Image{
						Image: &storage.ContainerImage{
							Id:   "nginx",
							Name: &storage.ImageName{FullName: "nginx"},
						},
					},
					Policy: &storage.Policy{
						Id:                 "policy2",
						Name:               "test Policy 2",
						Description:        "test description 2",
						Remediation:        "test remediation 2",
						Severity:           storage.Severity_LOW_SEVERITY,
						EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
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
					Entity: &storage.Alert_Deployment_{
						Deployment: &storage.Alert_Deployment{
							Id:        "deployment",
							Name:      "test-deployment",
							Type:      "deployment",
							Namespace: "default",
						},
					},
					Policy: &storage.Policy{
						Id:                 "policy2",
						Name:               "test Policy 2",
						Description:        "test description 2",
						Remediation:        "test remediation 2",
						Severity:           storage.Severity_LOW_SEVERITY,
						EnforcementActions: []storage.EnforcementAction{storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT},
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
			expectedOutput: &Result{
				Results: []EntityResult{
					{
						Metadata: EntityMetadata{
							ID: "deployment",
							AdditionalInfo: map[string]string{
								"name":      "test-deployment",
								"type":      "deployment",
								"namespace": "default",
							},
						},
						ViolatedPolicies: []Policy{
							{
								Name:        "test Policy 1",
								Severity:    "HIGH",
								Description: "test description 1",
								Violation:   []string{"test violation 1", "test violation 2"},
								Remediation: "test remediation 1",
							},
							{
								Name:         "test Policy 2",
								Severity:     "LOW",
								Description:  "test description 2",
								Violation:    []string{"test violation 1", "test violation 2"},
								Remediation:  "test remediation 2",
								FailingCheck: true,
							},
						},
						Summary: map[string]int{
							"TOTAL":    2,
							"LOW":      1,
							"MEDIUM":   0,
							"HIGH":     1,
							"CRITICAL": 0,
						},
					},
					{
						Metadata: EntityMetadata{
							ID: "nginx",
							AdditionalInfo: map[string]string{
								"name": "nginx",
								"type": "image",
							},
						},
						ViolatedPolicies: []Policy{
							{
								Name:        "test Policy 1",
								Severity:    "HIGH",
								Description: "test description 1",
								Violation:   []string{"test violation 1", "test violation 2"},
								Remediation: "test remediation 1",
							},
							{
								Name:         "test Policy 2",
								Severity:     "LOW",
								Description:  "test description 2",
								Violation:    []string{"test violation 1", "test violation 2"},
								Remediation:  "test remediation 2",
								FailingCheck: true,
							},
						},
						Summary: map[string]int{
							"TOTAL":    2,
							"LOW":      1,
							"MEDIUM":   0,
							"HIGH":     1,
							"CRITICAL": 0,
						},
					},
				},
				Summary: map[string]int{
					"TOTAL":    4,
					"LOW":      2,
					"MEDIUM":   0,
					"HIGH":     2,
					"CRITICAL": 0,
				},
			},
		},
		"policy violations with optional fields being empty": {
			alerts: []*storage.Alert{
				{
					Policy: &storage.Policy{
						Id:          "policy1",
						Name:        "test Policy 1",
						Description: "",
						Remediation: "",
						Severity:    storage.Severity_HIGH_SEVERITY,
					},
					Violations: nil,
				},
			},
			expectedOutput: &Result{
				Results: []EntityResult{
					{
						Summary: map[string]int{
							"TOTAL":    1,
							"LOW":      0,
							"MEDIUM":   0,
							"HIGH":     1,
							"CRITICAL": 0,
						},
						ViolatedPolicies: []Policy{
							{
								Name:        "test Policy 1",
								Severity:    "HIGH",
								Description: "",
								Violation:   []string{},
								Remediation: "",
							},
						},
						Metadata: EntityMetadata{ID: "unknown"},
					},
				},
				Summary: map[string]int{
					"TOTAL":    1,
					"LOW":      0,
					"MEDIUM":   0,
					"HIGH":     1,
					"CRITICAL": 0,
				},
			},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			policySummary := NewPolicySummaryForPrinting(c.alerts, storage.EnforcementAction_FAIL_BUILD_ENFORCEMENT)
			assert.Equal(t, c.expectedOutput, policySummary)
		})
	}
}
