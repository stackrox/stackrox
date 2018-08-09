package getters

import "github.com/stackrox/rox/central/dnrintegration"

// MockDNRIntegration is a mock integration with D&R.
type MockDNRIntegration struct {
	ExpectedNamespace   string
	ExpectedServiceName string
	MockAlerts          []dnrintegration.PolicyAlert
	MockError           error
}

// Test panics.
func (MockDNRIntegration) Test() error {
	panic("implement me")
}

// Alerts returns the values set in the mock.
func (m MockDNRIntegration) Alerts(namespace, serviceName string) ([]dnrintegration.PolicyAlert, error) {
	if m.ExpectedNamespace != "" && m.ExpectedNamespace != namespace {
		panic("Alerts called with wrong namespace")
	}
	if m.ExpectedServiceName != "" && m.ExpectedServiceName != serviceName {
		panic("Alerts called with wrong service name")
	}
	return m.MockAlerts, m.MockError
}

// MockDNRIntegrationGetter is a mock impelentatino of DNRIntegrationGetter.
type MockDNRIntegrationGetter struct {
	MockDNRIntegration dnrintegration.DNRIntegration
	Exists             bool
	Err                error
}

// ForCluster returns the set values set in the mock.
func (m MockDNRIntegrationGetter) ForCluster(clusterID string) (dnrintegration.DNRIntegration, bool, error) {
	return m.MockDNRIntegration, m.Exists, m.Err
}
