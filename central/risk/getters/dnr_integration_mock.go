package getters

import "github.com/stackrox/rox/central/dnrintegration"

// MockDNRIntegration is a mock integration with D&R.
type MockDNRIntegration struct {
	ExpectedClusterID   string
	ExpectedNamespace   string
	ExpectedServiceName string
	MockAlerts          []dnrintegration.PolicyAlert
	MockBaseURL         string
	MockError           error
}

// Alerts returns the values set in the mock.
func (m MockDNRIntegration) Alerts(clusterID, namespace, serviceName string) (dnrintegration.AlertsWithMetadata, error) {
	if m.ExpectedClusterID != "" && m.ExpectedClusterID != clusterID {
		panic("Alerts called with wrong cluster id")
	}
	if m.ExpectedNamespace != "" && m.ExpectedNamespace != namespace {
		panic("Alerts called with wrong namespace")
	}
	if m.ExpectedServiceName != "" && m.ExpectedServiceName != serviceName {
		panic("Alerts called with wrong service name")
	}
	return dnrintegration.AlertsWithMetadata{Alerts: m.MockAlerts, BaseURL: m.MockBaseURL}, m.MockError
}

// MockDNRIntegrationGetter is a mock implementation of DNRIntegrationGetter.
type MockDNRIntegrationGetter struct {
	MockDNRIntegration dnrintegration.DNRIntegration
	Exists             bool
}

// ForCluster returns the set values set in the mock.
func (m MockDNRIntegrationGetter) ForCluster(clusterID string) (dnrintegration.DNRIntegration, bool) {
	return m.MockDNRIntegration, m.Exists
}
