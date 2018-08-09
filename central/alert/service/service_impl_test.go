package service

import (
	"testing"

	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestGroupAlerts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    []*v1.ListAlert
		expected *v1.GetAlertsGroupResponse
	}{
		{
			name: "one category",
			input: []*v1.ListAlert{
				{
					Id: "id1",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Id:         "id1",
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 300},
				},
				{
					Id: "id2",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Id:         "id2",
						Name:       "policy2",
						Severity:   v1.Severity_HIGH_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 200},
				},
				{
					Id: "id3",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Id:         "id1",
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 100},
				},
			},
			expected: &v1.GetAlertsGroupResponse{
				AlertsByPolicies: []*v1.GetAlertsGroupResponse_PolicyGroup{
					{
						Policy: &v1.ListAlertPolicy{
							Categories: []string{"Image Assurance"},
							Id:         "id1",
							Name:       "policy1",
							Severity:   v1.Severity_LOW_SEVERITY,
						},
						NumAlerts: 2,
					},
					{
						Policy: &v1.ListAlertPolicy{
							Categories: []string{"Image Assurance"},
							Id:         "id2",
							Name:       "policy2",
							Severity:   v1.Severity_HIGH_SEVERITY,
						},
						NumAlerts: 1,
					},
				},
			},
		},
		{
			name: "multiple categories",
			input: []*v1.ListAlert{
				{
					Id: "id1",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Id:         "id1",
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 300},
				},
				{
					Id: "id2",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance", "Privileges Capabilities"},
						Id:         "id2",
						Name:       "policy2",
						Severity:   v1.Severity_HIGH_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 200},
				},
				{
					Id: "id3",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Container Configuration"},
						Id:         "id30",
						Name:       "policy30",
						Severity:   v1.Severity_CRITICAL_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 150},
				},
				{
					Id: "id4",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Id:         "id1",
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 100},
				},
			},
			expected: &v1.GetAlertsGroupResponse{
				AlertsByPolicies: []*v1.GetAlertsGroupResponse_PolicyGroup{
					{
						Policy: &v1.ListAlertPolicy{
							Categories: []string{"Image Assurance"},
							Id:         "id1",
							Name:       "policy1",
							Severity:   v1.Severity_LOW_SEVERITY,
						},
						NumAlerts: 2,
					},
					{
						Policy: &v1.ListAlertPolicy{
							Categories: []string{"Image Assurance", "Privileges Capabilities"},
							Id:         "id2",
							Name:       "policy2",
							Severity:   v1.Severity_HIGH_SEVERITY,
						},
						NumAlerts: 1,
					},
					{
						Policy: &v1.ListAlertPolicy{
							Categories: []string{"Container Configuration"},
							Id:         "id30",
							Name:       "policy30",
							Severity:   v1.Severity_CRITICAL_SEVERITY,
						},
						NumAlerts: 1,
					},
				},
			},
		},
	}

	alertService := &serviceImpl{}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := alertService.groupAlerts(c.input)

			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestCountAlerts(t *testing.T) {
	t.Parallel()

	// cases have the same alert inputs, but differ in group by function.
	cases := []struct {
		name        string
		input       []*v1.ListAlert
		groupByFunc func(*v1.ListAlert) []string
		expected    *v1.GetAlertsCountsResponse
	}{
		{
			name: "not grouped",
			input: []*v1.ListAlert{
				{
					Id: "id1",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 300},
				},
				{
					Id: "id2",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Container Configuration"},
						Name:       "policy2",
						Severity:   v1.Severity_CRITICAL_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 200},
				},
				{
					Id: "id3",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 130},
				},
				{
					Id: "id4",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Privileges Capabilities"},
						Name:       "policy3",
						Severity:   v1.Severity_MEDIUM_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 120},
				},
				{
					Id: "id5",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance", "Container Configuration"},
						Name:       "policy4",
						Severity:   v1.Severity_HIGH_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 120},
				},
				{
					Id: "id6",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance", "Container Configuration"},
						Name:       "policy4",
						Severity:   v1.Severity_HIGH_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 110},
				},
			},
			groupByFunc: groupByFuncs[v1.GetAlertsCountsRequest_UNSET],
			expected: &v1.GetAlertsCountsResponse{
				Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
					{
						Group: "",
						Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
							{
								Severity: v1.Severity_LOW_SEVERITY,
								Count:    2,
							},
							{
								Severity: v1.Severity_MEDIUM_SEVERITY,
								Count:    1,
							},
							{
								Severity: v1.Severity_HIGH_SEVERITY,
								Count:    2,
							},
							{
								Severity: v1.Severity_CRITICAL_SEVERITY,
								Count:    1,
							},
						},
					},
				},
			},
		},
		{
			name: "group by category",
			input: []*v1.ListAlert{
				{
					Id: "id1",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 300},
				},
				{
					Id: "id2",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Container Configuration"},
						Name:       "policy2",
						Severity:   v1.Severity_CRITICAL_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 200},
				},
				{
					Id: "id3",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 130},
				},
				{
					Id: "id4",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Privileges Capabilities"},
						Name:       "policy3",
						Severity:   v1.Severity_MEDIUM_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 120},
				},
				{
					Id: "id5",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance", "Container Configuration"},
						Name:       "policy4",
						Severity:   v1.Severity_HIGH_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 120},
				},
				{
					Id: "id6",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance", "Container Configuration"},
						Name:       "policy4",
						Severity:   v1.Severity_HIGH_SEVERITY,
					},
					Time: &timestamp.Timestamp{Seconds: 110},
				},
			},
			groupByFunc: groupByFuncs[v1.GetAlertsCountsRequest_CATEGORY],
			expected: &v1.GetAlertsCountsResponse{
				Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
					{
						Group: "Container Configuration",
						Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
							{
								Severity: v1.Severity_HIGH_SEVERITY,
								Count:    2,
							},
							{
								Severity: v1.Severity_CRITICAL_SEVERITY,
								Count:    1,
							},
						},
					},
					{
						Group: "Image Assurance",
						Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
							{
								Severity: v1.Severity_LOW_SEVERITY,
								Count:    2,
							},
							{
								Severity: v1.Severity_HIGH_SEVERITY,
								Count:    2,
							},
						},
					},
					{
						Group: "Privileges Capabilities",
						Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
							{
								Severity: v1.Severity_MEDIUM_SEVERITY,
								Count:    1,
							},
						},
					},
				},
			},
		},
		{
			name: "group by cluster",
			input: []*v1.ListAlert{
				{
					Id: "id1",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Deployment: &v1.ListAlertDeployment{
						ClusterName: "test",
					},
					Time: &timestamp.Timestamp{Seconds: 300},
				},
				{
					Id: "id2",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Container Configuration"},
						Name:       "policy2",
						Severity:   v1.Severity_CRITICAL_SEVERITY,
					},
					Deployment: &v1.ListAlertDeployment{
						ClusterName: "test",
					},
					Time: &timestamp.Timestamp{Seconds: 200},
				},
				{
					Id: "id3",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance"},
						Name:       "policy1",
						Severity:   v1.Severity_LOW_SEVERITY,
					},
					Deployment: &v1.ListAlertDeployment{
						ClusterName: "prod",
					},
					Time: &timestamp.Timestamp{Seconds: 130},
				},
				{
					Id: "id4",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Privileges Capabilities"},
						Name:       "policy3",
						Severity:   v1.Severity_MEDIUM_SEVERITY,
					},
					Deployment: &v1.ListAlertDeployment{
						ClusterName: "prod",
					},
					Time: &timestamp.Timestamp{Seconds: 120},
				},
				{
					Id: "id5",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance", "Container Configuration"},
						Name:       "policy4",
						Severity:   v1.Severity_HIGH_SEVERITY,
					},
					Deployment: &v1.ListAlertDeployment{
						ClusterName: "prod",
					},
					Time: &timestamp.Timestamp{Seconds: 120},
				},
				{
					Id: "id6",
					Policy: &v1.ListAlertPolicy{
						Categories: []string{"Image Assurance", "Container Configuration"},
						Name:       "policy4",
						Severity:   v1.Severity_HIGH_SEVERITY,
					},
					Deployment: &v1.ListAlertDeployment{
						ClusterName: "test",
					},
					Time: &timestamp.Timestamp{Seconds: 110},
				},
			},
			groupByFunc: groupByFuncs[v1.GetAlertsCountsRequest_CLUSTER],
			expected: &v1.GetAlertsCountsResponse{
				Groups: []*v1.GetAlertsCountsResponse_AlertGroup{
					{
						Group: "prod",
						Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
							{
								Severity: v1.Severity_LOW_SEVERITY,
								Count:    1,
							},
							{
								Severity: v1.Severity_MEDIUM_SEVERITY,
								Count:    1,
							},
							{
								Severity: v1.Severity_HIGH_SEVERITY,
								Count:    1,
							},
						},
					},
					{
						Group: "test",
						Counts: []*v1.GetAlertsCountsResponse_AlertGroup_AlertCounts{
							{
								Severity: v1.Severity_LOW_SEVERITY,
								Count:    1,
							},
							{
								Severity: v1.Severity_HIGH_SEVERITY,
								Count:    1,
							},
							{
								Severity: v1.Severity_CRITICAL_SEVERITY,
								Count:    1,
							},
						},
					},
				},
			},
		},
	}

	alertService := &serviceImpl{}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := alertService.countAlerts(c.input, c.groupByFunc)

			assert.Equal(t, c.expected, actual)
		})
	}
}

func TestGenerateTimeseries(t *testing.T) {
	alerts := []*v1.ListAlert{
		{
			Id: "id1",
			Time: &timestamp.Timestamp{
				Seconds: 1,
			},
			Stale: true,
			MarkedStale: &timestamp.Timestamp{
				Seconds: 8,
			},
			Deployment: &v1.ListAlertDeployment{ClusterName: "dev"},
			Policy:     &v1.ListAlertPolicy{Severity: v1.Severity_CRITICAL_SEVERITY},
		},
		{
			Id: "id2",
			Time: &timestamp.Timestamp{
				Seconds: 6,
			},
			Deployment: &v1.ListAlertDeployment{ClusterName: "dev"},
			Policy:     &v1.ListAlertPolicy{Severity: v1.Severity_HIGH_SEVERITY},
		},
		{
			Id: "id3",
			Time: &timestamp.Timestamp{
				Seconds: 1,
			},
			Stale: true,
			MarkedStale: &timestamp.Timestamp{
				Seconds: 8,
			},
			Deployment: &v1.ListAlertDeployment{ClusterName: "prod"},
			Policy:     &v1.ListAlertPolicy{Severity: v1.Severity_LOW_SEVERITY},
		},
		{
			Id: "id4",
			Time: &timestamp.Timestamp{
				Seconds: 6,
			},
			Deployment: &v1.ListAlertDeployment{ClusterName: "prod"},
			Policy:     &v1.ListAlertPolicy{Severity: v1.Severity_MEDIUM_SEVERITY},
		},
	}

	expectedEvents := map[string]map[v1.Severity][]*v1.AlertEvent{
		"dev": {
			v1.Severity_CRITICAL_SEVERITY: []*v1.AlertEvent{
				{
					Time: 1000,
					Id:   "id1",
					Type: v1.Type_CREATED,
				},
				{
					Time: 8000,
					Id:   "id1",
					Type: v1.Type_REMOVED,
				},
			},
			v1.Severity_HIGH_SEVERITY: []*v1.AlertEvent{
				{
					Time: 6000,
					Id:   "id2",
					Type: v1.Type_CREATED,
				},
			},
		},
		"prod": {
			v1.Severity_LOW_SEVERITY: []*v1.AlertEvent{
				{
					Time: 1000,
					Id:   "id3",
					Type: v1.Type_CREATED,
				},
				{
					Time: 8000,
					Id:   "id3",
					Type: v1.Type_REMOVED,
				},
			},
			v1.Severity_MEDIUM_SEVERITY: []*v1.AlertEvent{
				{
					Time: 6000,
					Id:   "id4",
					Type: v1.Type_CREATED,
				},
			},
		},
	}
	assert.Empty(t, getGroupToAlertEvents(nil))
	assert.Equal(t, expectedEvents, getGroupToAlertEvents(alerts))
}
