package risk

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/central/dnrintegration"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type mockDNRIntegration struct {
	expectedNamespace   string
	expectedServiceName string
	mockAlerts          []dnrintegration.PolicyAlert
	mockError           error
}

func (mockDNRIntegration) Test() error {
	panic("implement me")
}

func (m mockDNRIntegration) Alerts(namespace, serviceName string) ([]dnrintegration.PolicyAlert, error) {
	if m.expectedNamespace != "" && m.expectedNamespace != namespace {
		panic("Alerts called with wrong namespace")
	}
	if m.expectedServiceName != "" && m.expectedServiceName != serviceName {
		panic("Alerts called with wrong service name")
	}
	return m.mockAlerts, m.mockError
}

type mockDNRIntegrationGetter struct {
	mockDNRIntegration dnrintegration.DNRIntegration
	exists             bool
	err                error
}

func (m mockDNRIntegrationGetter) ForCluster(clusterID string) (dnrintegration.DNRIntegration, bool, error) {
	return m.mockDNRIntegration, m.exists, m.err
}

func TestDNRAlerts(t *testing.T) {
	const fakeNamespace = "FAKENAMESPACE"
	const fakeServiceName = "FAKESERVICENAME"

	cases := []struct {
		name string

		integrationExists       bool
		integrationRetrievalErr error

		expectedNamespace   string
		expectedServiceName string
		mockAlerts          []dnrintegration.PolicyAlert
		mockError           error

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
			expectedNamespace:   fakeNamespace,
			expectedServiceName: fakeServiceName,
			mockAlerts:          make([]dnrintegration.PolicyAlert, 0),
			deployment: &v1.Deployment{
				Name:      fakeServiceName,
				Namespace: fakeNamespace,
			},
			expectedResult: nil,
		},
		{
			name:                "Couple of alerts",
			integrationExists:   true,
			expectedNamespace:   fakeNamespace,
			expectedServiceName: fakeServiceName,
			mockAlerts: []dnrintegration.PolicyAlert{
				{PolicyName: "FakePolicy0", SeverityWord: "CRITICAL", SeverityScore: 100},
				{PolicyName: "FakePolicy1", SeverityWord: "MEDIUM", SeverityScore: 50},
			},
			deployment: &v1.Deployment{
				Name:      fakeServiceName,
				Namespace: fakeNamespace,
			},
			expectedResult: &v1.Risk_Result{
				Name:    "Runtime Alerts",
				Factors: []string{"FakePolicy0 (Severity: CRITICAL)", "FakePolicy1 (Severity: MEDIUM)"},
				Score:   1.5,
			},
		},
		{
			name:                "Tons of alerts",
			integrationExists:   true,
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
			deployment: &v1.Deployment{
				Name:      fakeServiceName,
				Namespace: fakeNamespace,
			},
			expectedResult: &v1.Risk_Result{
				Name: "Runtime Alerts",
				Factors: []string{
					"FakePolicy0 (Severity: CRITICAL) (10+ x)",
					"FakePolicy3 (Severity: SUPER CRITICAL)",
					"FakePolicy1 (Severity: MEDIUM)",
					"FakePolicy2 (Severity: LOW)",
					"FakePolicy5 (Severity: LOW) (2x)",
					"2 Other Alerts",
				},
				Score: 2.0,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mult := newDNRAlertMultiplier(&mockDNRIntegrationGetter{
				mockDNRIntegration: &mockDNRIntegration{
					expectedNamespace:   c.expectedNamespace,
					expectedServiceName: c.expectedServiceName,
					mockAlerts:          c.mockAlerts,
					mockError:           c.mockError,
				},
				exists: c.integrationExists,
			})
			result := mult.Score(c.deployment)
			assert.Equal(t, c.expectedResult, result)
		})
	}
}
