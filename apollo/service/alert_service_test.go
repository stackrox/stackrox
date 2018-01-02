package service

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"
)

func TestGroupAlerts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    []*v1.Alert
		expected *v1.GetAlertsGroupResponse
	}{
		{
			name: "one category",
			input: []*v1.Alert{
				{
					Id: "id1",
					Policy: &v1.Policy{
						Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 300},
				},
				{
					Id: "id2",
					Policy: &v1.Policy{
						Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
						Name:       "policy2",
						Severity:   v1.Severity_HIGH_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 200},
				},
				{
					Id: "id3",
					Policy: &v1.Policy{
						Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 100},
				},
			},
			expected: &v1.GetAlertsGroupResponse{
				ByCategory: []*v1.GetAlertsGroupResponse_CategoryGroup{
					{
						Category: v1.Policy_Category_IMAGE_ASSURANCE,
						ByPolicy: []*v1.GetAlertsGroupResponse_PolicyGroup{
							{
								Policy: &v1.Policy{
									Name:       "policy1",
									Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
									Severity:   v1.Severity_LOW_SEVERITY,
								},
								NumAlerts: 2,
							},
							{
								Policy: &v1.Policy{
									Name:       "policy2",
									Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
									Severity:   v1.Severity_HIGH_SEVERITY,
								},
								NumAlerts: 1,
							},
						},
					},
				},
			},
		},
		{
			name: "multiple categories",
			input: []*v1.Alert{
				{
					Id: "id1",
					Policy: &v1.Policy{
						Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 300},
				},
				{
					Id: "id2",
					Policy: &v1.Policy{
						Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE, v1.Policy_Category_PRIVILEGES_CAPABILITIES},
						Name:       "policy2",
						Severity:   v1.Severity_HIGH_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 200},
				},
				{
					Id: "id3",
					Policy: &v1.Policy{
						Categories: []v1.Policy_Category{v1.Policy_Category_CONTAINER_CONFIGURATION},
						Name:       "policy10",
						Severity:   v1.Severity_CRITICAL_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 150},
				},
				{
					Id: "id4",
					Policy: &v1.Policy{
						Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 100},
				},
			},
			expected: &v1.GetAlertsGroupResponse{
				ByCategory: []*v1.GetAlertsGroupResponse_CategoryGroup{
					{
						Category: v1.Policy_Category_IMAGE_ASSURANCE,
						ByPolicy: []*v1.GetAlertsGroupResponse_PolicyGroup{
							{
								Policy: &v1.Policy{
									Name:       "policy1",
									Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE},
									Severity:   v1.Severity_LOW_SEVERITY,
								},
								NumAlerts: 2,
							},
							{
								Policy: &v1.Policy{
									Name:       "policy2",
									Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE, v1.Policy_Category_PRIVILEGES_CAPABILITIES},
									Severity:   v1.Severity_HIGH_SEVERITY,
								},
								NumAlerts: 1,
							},
						},
					},
					{
						Category: v1.Policy_Category_PRIVILEGES_CAPABILITIES,
						ByPolicy: []*v1.GetAlertsGroupResponse_PolicyGroup{
							{
								Policy: &v1.Policy{
									Name:       "policy2",
									Categories: []v1.Policy_Category{v1.Policy_Category_IMAGE_ASSURANCE, v1.Policy_Category_PRIVILEGES_CAPABILITIES},
									Severity:   v1.Severity_HIGH_SEVERITY,
								},
								NumAlerts: 1,
							},
						},
					},
					{
						Category: v1.Policy_Category_CONTAINER_CONFIGURATION,
						ByPolicy: []*v1.GetAlertsGroupResponse_PolicyGroup{
							{
								Policy: &v1.Policy{
									Name:       "policy10",
									Categories: []v1.Policy_Category{v1.Policy_Category_CONTAINER_CONFIGURATION},
									Severity:   v1.Severity_CRITICAL_SEVERITY,
								},
								NumAlerts: 1,
							},
						},
					},
				},
			},
		},
	}

	alertService := &AlertService{}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := alertService.groupAlerts(c.input)

			assert.Equal(t, actual, c.expected)
		})
	}
}
