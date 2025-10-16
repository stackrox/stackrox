package deployment

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
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
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 1",
					}.Build(),
				}.Build(),
			},
			expected: storage.Risk_Result_builder{
				Name: PolicyViolationsHeading,
				Factors: []*storage.Risk_Result_Factor{
					storage.Risk_Result_Factor_builder{Message: "Policy 1 (severity: Critical)"}.Build(),
				},
				Score: 1.96,
			}.Build(),
		},
		{
			name: "Two critical",
			alerts: []*storage.ListAlert{
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 1",
					}.Build(),
				}.Build(),
			},
			expected: storage.Risk_Result_builder{
				Name: PolicyViolationsHeading,
				Factors: []*storage.Risk_Result_Factor{
					storage.Risk_Result_Factor_builder{Message: "Policy 1 (severity: Critical)"}.Build(),
				},
				Score: 1.96,
			}.Build(),
		},
		{
			name: "Mix of severities (1)",
			alerts: []*storage.ListAlert{
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_HIGH_SEVERITY,
						Name:     "Policy 1",
					}.Build(),
				}.Build(),
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_MEDIUM_SEVERITY,
						Name:     "Policy 2",
					}.Build(),
				}.Build(),
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_LOW_SEVERITY,
						Name:     "Policy 3",
					}.Build(),
				}.Build(),
			},
			expected: storage.Risk_Result_builder{
				Name: PolicyViolationsHeading,
				Factors: []*storage.Risk_Result_Factor{
					storage.Risk_Result_Factor_builder{Message: "Policy 1 (severity: High)"}.Build(),
					storage.Risk_Result_Factor_builder{Message: "Policy 2 (severity: Medium)"}.Build(),
					storage.Risk_Result_Factor_builder{Message: "Policy 3 (severity: Low)"}.Build(),
				},
				Score: 1.84,
			}.Build(),
		},
		{
			name: "Mix of severities (2)",
			alerts: []*storage.ListAlert{
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 1",
					}.Build(),
				}.Build(),
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_HIGH_SEVERITY,
						Name:     "Policy 2",
					}.Build(),
				}.Build(),
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_LOW_SEVERITY,
						Name:     "Policy 3",
					}.Build(),
				}.Build(),
			},
			expected: storage.Risk_Result_builder{
				Name: PolicyViolationsHeading,
				Factors: []*storage.Risk_Result_Factor{
					storage.Risk_Result_Factor_builder{Message: "Policy 1 (severity: Critical)"}.Build(),
					storage.Risk_Result_Factor_builder{Message: "Policy 2 (severity: High)"}.Build(),
					storage.Risk_Result_Factor_builder{Message: "Policy 3 (severity: Low)"}.Build(),
				},
				Score: 2.56,
			}.Build(),
		},
		{
			name: "Don't include stale alerts",
			alerts: []*storage.ListAlert{
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 3",
					}.Build(),
				}.Build(),
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_HIGH_SEVERITY,
						Name:     "Policy 2",
					}.Build(),
				}.Build(),
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_LOW_SEVERITY,
						Name:     "Policy 1",
					}.Build(),
				}.Build(),
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_CRITICAL_SEVERITY,
						Name:     "Policy Don't Show Me!",
					}.Build(),
					State: storage.ViolationState_RESOLVED,
				}.Build(),
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_HIGH_SEVERITY,
						Name:     "Policy Don't Show Me!",
					}.Build(),
					State: storage.ViolationState_RESOLVED,
				}.Build(),
				storage.ListAlert_builder{
					Policy: storage.ListAlertPolicy_builder{
						Severity: storage.Severity_LOW_SEVERITY,
						Name:     "Policy Don't Show Me!",
					}.Build(),
					State: storage.ViolationState_RESOLVED,
				}.Build(),
			},
			expected: storage.Risk_Result_builder{
				Name: PolicyViolationsHeading,
				Factors: []*storage.Risk_Result_Factor{
					storage.Risk_Result_Factor_builder{Message: "Policy 3 (severity: Critical)"}.Build(),
					storage.Risk_Result_Factor_builder{Message: "Policy 2 (severity: High)"}.Build(),
					storage.Risk_Result_Factor_builder{Message: "Policy 1 (severity: Low)"}.Build(),
				},
				Score: 2.56,
			}.Build(),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mult := NewViolations(&getters.MockAlertsSearcher{
				Alerts: c.alerts,
			})
			deployment := multipliers.GetMockDeployment()
			result := mult.Score(context.Background(), deployment, nil)
			protoassert.Equal(t, c.expected, result)
		})
	}
}
