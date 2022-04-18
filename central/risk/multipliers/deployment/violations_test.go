package deployment

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestViolationsScore(t *testing.T) {
	cases := []struct {
		name     string
		alerts   []*storage.ListAlert
		expected *storage.Risk_Result
	}{
		{
			name:     "No alerts",
			alerts:   nil,
			expected: nil,
		},
		{
			name: "One critical",
			alerts: []*storage.ListAlert{
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 1",
					},
				},
			},
			expected: &storage.Risk_Result{
				Name: PolicyViolationsHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Policy 1 (severity: Critical)"},
				},
				Score: 1.96,
			},
		},
		{
			name: "Two critical",
			alerts: []*storage.ListAlert{
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 1",
					},
				},
			},
			expected: &storage.Risk_Result{
				Name: PolicyViolationsHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Policy 1 (severity: Critical)"},
				},
				Score: 1.96,
			},
		},
		{
			name: "Mix of severities (1)",
			alerts: []*storage.ListAlert{
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_HIGH_SEVERITY,
						Name:     "Policy 1",
					},
				},
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_MEDIUM_SEVERITY,
						Name:     "Policy 2",
					},
				},
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_LOW_SEVERITY,
						Name:     "Policy 3",
					},
				},
			},
			expected: &storage.Risk_Result{
				Name: PolicyViolationsHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Policy 1 (severity: High)"},
					{Message: "Policy 2 (severity: Medium)"},
					{Message: "Policy 3 (severity: Low)"},
				},
				Score: 1.84,
			},
		},
		{
			name: "Mix of severities (2)",
			alerts: []*storage.ListAlert{
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 1",
					},
				},
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_HIGH_SEVERITY,
						Name:     "Policy 2",
					},
				},
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_LOW_SEVERITY,
						Name:     "Policy 3",
					},
				},
			},
			expected: &storage.Risk_Result{
				Name: PolicyViolationsHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Policy 1 (severity: Critical)"},
					{Message: "Policy 2 (severity: High)"},
					{Message: "Policy 3 (severity: Low)"},
				},
				Score: 2.56,
			},
		},
		{
			name: "Don't include stale alerts",
			alerts: []*storage.ListAlert{
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 3",
					},
				},
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_HIGH_SEVERITY,
						Name:     "Policy 2",
					},
				},
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_LOW_SEVERITY,
						Name:     "Policy 1",
					},
				},
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Name:     "Policy Don't Show Me!",
					},
					State: storage.ViolationState_RESOLVED,
				},
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_HIGH_SEVERITY,
						Name:     "Policy Don't Show Me!",
					},
					State: storage.ViolationState_RESOLVED,
				},
				{
					Policy: &storage.ListAlertPolicy{
						Severity: storage.Severity_LOW_SEVERITY,
						Name:     "Policy Don't Show Me!",
					},
					State: storage.ViolationState_RESOLVED,
				},
			},
			expected: &storage.Risk_Result{
				Name: PolicyViolationsHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Policy 3 (severity: Critical)"},
					{Message: "Policy 2 (severity: High)"},
					{Message: "Policy 1 (severity: Low)"},
				},
				Score: 2.56,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mult := NewViolations(&getters.MockAlertsSearcher{
				Alerts: c.alerts,
			})
			deployment := multipliers.GetMockDeployment()
			result := mult.Score(context.Background(), deployment, nil)
			assert.Equal(t, c.expected, result)
		})
	}
}
