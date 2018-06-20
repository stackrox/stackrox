package risk

import (
	"fmt"
	"testing"

	"bitbucket.org/stack-rox/apollo/central/search"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

type mockGetter struct {
	alerts []*v1.Alert
}

// GetAlerts supports a limited set of request parameters.
// It only needs to be as specific as the production code.
func (m mockGetter) GetAlerts(req *v1.ListAlertsRequest) (alerts []*v1.Alert, err error) {
	parsedRequest, err := search.ParseRawQuery(req.GetQuery())
	if err != nil {
		return nil, err
	}
	for _, a := range m.alerts {
		match := true
		staleValues := parsedRequest.Fields["alert.stale"].GetValues()
		if len(staleValues) != 0 {
			match = false
			for _, v := range staleValues {
				if fmt.Sprintf("%t", a.Stale) == v {
					match = true
				}
			}
		}
		if match {
			alerts = append(alerts, a)
		}
	}
	return
}

func TestViolationsScore(t *testing.T) {
	cases := []struct {
		name     string
		alerts   []*v1.Alert
		expected *v1.Risk_Result
	}{
		{
			name:     "No alerts",
			alerts:   nil,
			expected: nil,
		},
		{
			name: "One critical",
			alerts: []*v1.Alert{
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 1",
					},
				},
			},
			expected: &v1.Risk_Result{
				Name: policyViolationsHeading,
				Factors: []string{
					"Policy 1 (severity: Critical)",
				},
				Score: 1.2,
			},
		},
		{
			name: "Two critical",
			alerts: []*v1.Alert{
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 1",
					},
				},
			},
			expected: &v1.Risk_Result{
				Name: policyViolationsHeading,
				Factors: []string{
					"Policy 1 (severity: Critical)",
				},
				Score: 1.2,
			},
		},
		{
			name: "Mix of severities (1)",
			alerts: []*v1.Alert{
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_HIGH_SEVERITY,
						Name:     "Policy 1",
					},
				},
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_MEDIUM_SEVERITY,
						Name:     "Policy 2",
					},
				},
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_LOW_SEVERITY,
						Name:     "Policy 3",
					},
				},
			},
			expected: &v1.Risk_Result{
				Name: policyViolationsHeading,
				Factors: []string{
					"Policy 1 (severity: High)",
					"Policy 2 (severity: Medium)",
					"Policy 3 (severity: Low)",
				},
				Score: 1.3,
			},
		},
		{
			name: "Mix of severities (2)",
			alerts: []*v1.Alert{
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 1",
					},
				},
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_HIGH_SEVERITY,
						Name:     "Policy 2",
					},
				},
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_LOW_SEVERITY,
						Name:     "Policy 3",
					},
				},
			},
			expected: &v1.Risk_Result{
				Name: policyViolationsHeading,
				Factors: []string{
					"Policy 1 (severity: Critical)",
					"Policy 2 (severity: High)",
					"Policy 3 (severity: Low)",
				},
				Score: 1.4,
			},
		},
		{
			name: "Don't include stale alerts",
			alerts: []*v1.Alert{
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_CRITICAL_SEVERITY,
						Name:     "Policy 3",
					},
				},
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_HIGH_SEVERITY,
						Name:     "Policy 2",
					},
				},
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_LOW_SEVERITY,
						Name:     "Policy 1",
					},
				},
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_CRITICAL_SEVERITY,
						Name:     "Policy Don't Show Me!",
					},
					Stale: true,
				},
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_HIGH_SEVERITY,
						Name:     "Policy Don't Show Me!",
					},
					Stale: true,
				},
				{
					Policy: &v1.Policy{
						Severity: v1.Severity_LOW_SEVERITY,
						Name:     "Policy Don't Show Me!",
					},
					Stale: true,
				},
			},
			expected: &v1.Risk_Result{
				Name: policyViolationsHeading,
				Factors: []string{
					"Policy 3 (severity: Critical)",
					"Policy 2 (severity: High)",
					"Policy 1 (severity: Low)",
				},
				Score: 1.4,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mult := newViolationsMultiplier(&mockGetter{
				alerts: c.alerts,
			})
			deployment := getMockDeployment()
			result := mult.Score(deployment)
			assert.Equal(t, c.expected, result)
		})
	}
}
