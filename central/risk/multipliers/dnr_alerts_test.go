package multipliers

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/central/dnrintegration"
	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestDNRAlerts(t *testing.T) {
	const fakeClusterID = "FAKECLUSTERID"
	const fakeNamespace = "FAKENAMESPACE"
	const fakeServiceName = "FAKESERVICENAME"

	fakeDeployment := &v1.Deployment{
		ClusterId: fakeClusterID,
		Name:      fakeServiceName,
		Namespace: fakeNamespace,
	}

	cases := []struct {
		name string

		integrationExists       bool
		integrationRetrievalErr error

		expectedClusterID   string
		expectedNamespace   string
		expectedServiceName string
		mockAlerts          []dnrintegration.PolicyAlert
		mockError           error
		mockBaseURL         string

		deployment     *v1.Deployment
		expectedResult *v1.Risk_Result
	}{
		{
			name:              "No integration",
			integrationExists: false,
			expectedResult:    nil,
		},
		{
			name:                    "Error retrieving integration",
			integrationExists:       true,
			integrationRetrievalErr: errors.New("my fake error"),
			expectedResult:          nil,
		},
		{
			name:              "Error retrieving integration",
			integrationExists: true,
			mockError:         errors.New("my fake error"),
			expectedResult:    nil,
		},
		{
			name:                "No alerts",
			integrationExists:   true,
			expectedClusterID:   fakeClusterID,
			expectedNamespace:   fakeNamespace,
			expectedServiceName: fakeServiceName,
			mockAlerts:          make([]dnrintegration.PolicyAlert, 0),
			deployment:          fakeDeployment,
			expectedResult:      nil,
		},
		{
			name:                "Couple of alerts",
			integrationExists:   true,
			expectedClusterID:   fakeClusterID,
			expectedNamespace:   fakeNamespace,
			expectedServiceName: fakeServiceName,
			mockAlerts: []dnrintegration.PolicyAlert{
				{ID: "FAKEALERT", PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy1", SeverityWord: "MEDIUM", SeverityScore: 50},
			},
			mockBaseURL: "https://portal.stackrox",
			deployment:  fakeDeployment,
			expectedResult: &v1.Risk_Result{
				Name: "Runtime Alerts",
				Factors: []*v1.Risk_Result_Factor{
					{Message: "FakePolicy0 (Severity: CRITICAL)", Url: "https://portal.stackrox/main/incidents/alerts/alert/FAKEALERT"},
					{Message: "FakePolicy1 (Severity: MEDIUM)"}},
				Score: 1.5,
			},
		},
		{
			name:                "Tons of alerts",
			integrationExists:   true,
			expectedClusterID:   fakeClusterID,
			expectedNamespace:   fakeNamespace,
			expectedServiceName: fakeServiceName,
			mockAlerts: []dnrintegration.PolicyAlert{
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy1", SeverityWord: "MEDIUM", SeverityScore: 50},
				{PolicyName: "FakePolicy2", SeverityWord: "LOW", SeverityScore: 25},
				{PolicyName: "FakePolicy3", SeverityWord: "SUPER CRITICAL", SeverityScore: 200},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy4", SeverityWord: "LOW", SeverityScore: 20},
				{PolicyName: "FakePolicy5", SeverityWord: "LOW", SeverityScore: 15},
				{PolicyName: "FakePolicy5", SeverityWord: "LOW", SeverityScore: 15},
				{PolicyName: "FakePolicy6", SeverityWord: "LOW", SeverityScore: 10},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
			},
			deployment: fakeDeployment,
			expectedResult: &v1.Risk_Result{
				Name: "Runtime Alerts",
				Factors: []*v1.Risk_Result_Factor{
					{Message: "FakePolicy0 (Severity: CRITICAL) (10+ x)"},
					{Message: "FakePolicy3 (Severity: SUPER CRITICAL)"},
					{Message: "FakePolicy1 (Severity: MEDIUM)"},
					{Message: "FakePolicy2 (Severity: LOW)"},
					{Message: "FakePolicy5 (Severity: LOW) (2x)"},
					{Message: "2 Other Alerts"},
				},
				Score: 2.0,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mult := NewDNRAlert(&getters.MockDNRIntegrationGetter{
				MockDNRIntegration: &getters.MockDNRIntegration{
					ExpectedClusterID:   c.expectedClusterID,
					ExpectedNamespace:   c.expectedNamespace,
					ExpectedServiceName: c.expectedServiceName,
					MockAlerts:          c.mockAlerts,
					MockError:           c.mockError,
					MockBaseURL:         c.mockBaseURL,
				},
				Exists: c.integrationExists,
			})
			result := mult.Score(c.deployment)
			assert.Equal(t, c.expectedResult, result)
		})
	}
}
