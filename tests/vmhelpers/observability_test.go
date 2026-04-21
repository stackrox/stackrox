//go:build test

package vmhelpers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVMIndexMetricSeriesMirrorProductDefinitions(t *testing.T) {
	require.Equal(t, "rox_compliance_virtual_machine_relay_connections_accepted_total", MetricComplianceRelayConnectionsAcceptedTotal)
	require.Equal(t, "rox_compliance_virtual_machine_relay_index_reports_received_total", MetricComplianceRelayIndexReportsReceivedTotal)
	require.Equal(t, "rox_compliance_virtual_machine_relay_index_reports_sent_total", MetricComplianceRelayIndexReportsSentTotal)
	require.Equal(t, "rox_compliance_virtual_machine_relay_index_reports_mismatching_vsock_cid_total", MetricComplianceRelayIndexReportsMismatchingVsockTotal)
	require.Equal(t, "rox_compliance_virtual_machine_relay_sem_acquisition_failures_total", MetricComplianceRelaySemaphoreAcquisitionFailuresTotal)

	require.Equal(t, "rox_sensor_virtual_machine_index_reports_received_total", MetricSensorVMIndexReportsReceivedTotal)
	require.Equal(t, "rox_sensor_virtual_machine_index_reports_sent_total", MetricSensorVMIndexReportsSentTotal)
	require.Equal(t, "rox_sensor_virtual_machine_index_report_acks_received_total", MetricSensorVMIndexReportAcksReceivedTotal)
	require.Equal(t, "rox_sensor_virtual_machine_index_report_enqueue_blocked_total", MetricSensorVMIndexReportEnqueueBlockedTotal)
}

func moduleRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// TestCanonicalMetricGoSourceFilesDriftGuard reads product metric definitions and fails when
// required names/labels disappear from source (compliance relay, sensor VM index).
func TestCanonicalMetricGoSourceFilesDriftGuard(t *testing.T) {
	root := moduleRepoRoot(t)
	compliancePath := filepath.Join(root, "compliance/virtualmachines/relay/metrics/metrics.go")
	sensorPath := filepath.Join(root, "sensor/common/virtualmachine/metrics/metrics.go")

	cb, err := os.ReadFile(compliancePath)
	require.NoError(t, err, "read %s", compliancePath)
	compliance := string(cb)
	for _, sub := range []string{
		"virtual_machine_relay_connections_accepted_total",
		"virtual_machine_relay_index_reports_received_total",
		"virtual_machine_relay_index_reports_sent_total",
		"virtual_machine_relay_index_reports_mismatching_vsock_cid_total",
		"virtual_machine_relay_sem_acquisition_failures_total",
		`[]string{"failed"}`,
	} {
		require.Contains(t, compliance, sub, "compliance/virtualmachines/relay/metrics/metrics.go drift: missing %q", sub)
	}

	sb, err := os.ReadFile(sensorPath)
	require.NoError(t, err, "read %s", sensorPath)
	sensor := string(sb)
	for _, sub := range []string{
		"virtual_machine_index_reports_received_total",
		"virtual_machine_index_reports_sent_total",
		`[]string{"status"}`,
		`"central not ready"`,
		"virtual_machine_index_report_acks_received_total",
		`[]string{"action"}`,
		"virtual_machine_index_report_enqueue_blocked_total",
		"virtual_machine_index_report_blocking_enqueue_duration_milliseconds",
	} {
		require.Contains(t, sensor, sub, "sensor/common/virtualmachine/metrics/metrics.go drift: missing %q", sub)
	}

	require.Contains(t, MetricComplianceRelayConnectionsAcceptedTotal, "virtual_machine_relay_connections_accepted_total")
	require.Contains(t, MetricSensorVMIndexReportsReceivedTotal, "virtual_machine_index_reports_received_total")
}
