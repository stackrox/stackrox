//go:build test

package vmhelpers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func moduleRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// TestCanonicalMetricGoSourceFilesDriftGuard reads product metric definitions and fails when
// names/labels used by the VM scanning metrics assertions disappear from source.
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
