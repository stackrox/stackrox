//go:build test_e2e_vm

package tests

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/tests/testmetrics"
	"github.com/stackrox/rox/tests/vmhelpers"
	"github.com/stretchr/testify/require"
)

// complianceTarget returns the ScrapeTarget for the compliance container on the VM's node.
func (s *VMScanningSuite) complianceTarget(vmNodeName string) testmetrics.ScrapeTarget {
	t := testmetrics.ScrapeTarget{
		ComponentName: "compliance",
		Namespace:     namespaces.StackRox,
		LabelSelector: "app=collector",
		MetricsPort:   9091,
		MetricsPath:   "metrics",
	}
	if vmNodeName != "" {
		t.FieldSelector = "spec.nodeName=" + vmNodeName
	}
	return t
}

// sensorTarget returns the ScrapeTarget for the sensor deployment.
func (s *VMScanningSuite) sensorTarget() testmetrics.ScrapeTarget {
	return testmetrics.ScrapeTarget{
		ComponentName: "sensor",
		Namespace:     namespaces.StackRox,
		LabelSelector: "app=sensor",
		MetricsPort:   9090,
		MetricsPath:   "metrics",
	}
}

// complianceQueries returns the Query set for the compliance relay.
func complianceQueries() []testmetrics.Query {
	return []testmetrics.Query{
		{Name: vmhelpers.MetricComplianceRelayConnectionsAcceptedTotal},
		{Name: vmhelpers.MetricComplianceRelayIndexReportsReceivedTotal},
		{Name: vmhelpers.MetricComplianceRelayIndexReportsSentTotal, LabelFilter: `failed="false"`},
		{Name: vmhelpers.MetricComplianceRelayIndexReportsSentTotal, LabelFilter: `failed="true"`},
		{Name: vmhelpers.MetricComplianceRelayIndexReportsMismatchingVsockTotal},
		{Name: vmhelpers.MetricComplianceRelayIndexReportAcksReceivedTotal},
	}
}

// sensorQueries returns the Query set for the sensor VM index pipeline.
func sensorQueries() []testmetrics.Query {
	return []testmetrics.Query{
		{Name: vmhelpers.MetricSensorVMIndexReportsReceivedTotal},
		{Name: vmhelpers.MetricSensorVMIndexReportsSentTotal, LabelFilter: `status="` + vmhelpers.SensorIndexReportStatusSuccess + `"`},
		{Name: vmhelpers.MetricSensorVMIndexReportsSentTotal, LabelFilter: `status="` + vmhelpers.SensorIndexReportStatusError + `"`},
		{Name: vmhelpers.MetricSensorVMIndexReportsSentTotal, LabelFilter: `status="` + vmhelpers.SensorIndexReportStatusCentralNotReady + `"`},
		{Name: vmhelpers.MetricSensorVMIndexReportAcksReceivedTotal, LabelFilter: `action="ACK"`},
		{Name: vmhelpers.MetricSensorVMIndexReportEnqueueBlockedTotal},
	}
}

// collectStableMetrics scrapes compliance and sensor metrics until values stabilize.
// It returns two maps keyed by testmetrics.Key.
//
// Scraping uses the Kubernetes pods/proxy subresource which routes through the
// API server directly to the pod, bypassing Services and NetworkPolicies.
// The test still creates permissive NetworkPolicies and a compliance-metrics
// Service in ensureComplianceMetricsExposed as defence-in-depth.
func (s *VMScanningSuite) collectStableMetrics(ctx context.Context, vmNodeName string, compQ, senQ []testmetrics.Query) (compliance, sensor map[string]testmetrics.Value) {
	const (
		metricsTimeout  = 2 * time.Minute
		metricsPollWait = 10 * time.Second
		stableRounds    = 3
	)

	compTarget := s.complianceTarget(vmNodeName)
	senTarget := s.sensorTarget()
	// ponytail: empty transport defaults to proxy in scrapePod.
	// Switch back to testmetrics.TransportPortForward (with s.restCfg) if proxy proves unreliable.
	transport := testmetrics.TransportProxy
	restCfg := s.restCfg

	stableCfg := testmetrics.StableConfig{
		PollInterval: metricsPollWait,
		StableRounds: stableRounds,
		Logf:         s.logf,
	}

	compCtx, compCancel := context.WithTimeout(ctx, metricsTimeout)
	defer compCancel()
	compliance = testmetrics.PollUntilStable(compCtx, stableCfg, func(ctx context.Context) (map[string]testmetrics.Value, error) {
		return testmetrics.ScrapeComponent(ctx, s.k8sClient, compTarget, transport, restCfg, compQ)
	})

	senCtx, senCancel := context.WithTimeout(ctx, metricsTimeout)
	defer senCancel()
	sensor = testmetrics.PollUntilStable(senCtx, stableCfg, func(ctx context.Context) (map[string]testmetrics.Value, error) {
		return testmetrics.ScrapeComponent(ctx, s.k8sClient, senTarget, transport, restCfg, senQ)
	})

	return compliance, sensor
}

// assertPipelineMetrics collects stable metrics and asserts their values.
// vmNodeName must be non-empty so compliance metrics are scoped to the VM's local collector pod.
func (s *VMScanningSuite) assertPipelineMetrics(ctx context.Context, t require.TestingT, vmNodeName string) {
	require.NotEmpty(t, vmNodeName,
		"VM node name must be known before asserting pipeline metrics; "+
			"cluster-wide collector scraping is not supported because it conflates metrics from unrelated VMs")

	compTarget := s.complianceTarget(vmNodeName)
	s.logf("pipeline metrics: VM node=%q, compliance selector=%q field=%q",
		vmNodeName, compTarget.LabelSelector, compTarget.FieldSelector)

	err := testmetrics.FindServicePort(ctx, s.k8sClient, compTarget.Namespace, "app", "collector", compTarget.MetricsPort)
	require.NoError(t, err,
		"collector Service should expose compliance metrics port %d; the deployment may be missing the metrics port definition",
		compTarget.MetricsPort)

	cq := complianceQueries()
	sq := sensorQueries()
	comp, sen := s.collectStableMetrics(ctx, vmNodeName, cq, sq)

	get := func(src map[string]testmetrics.Value, q testmetrics.Query) testmetrics.Value {
		return src[testmetrics.Key(q)]
	}

	requirePositive := func(src map[string]testmetrics.Value, q testmetrics.Query, label string) {
		v := get(src, q)
		require.Truef(t, v.Found, "%s should be present in scraped metrics, but was not found", label)
		require.Greaterf(t, v.Val, float64(0), "%s should be > 0, but got %.0f", label, v.Val)
	}

	requireZero := func(src map[string]testmetrics.Value, q testmetrics.Query, label string) {
		v := get(src, q)
		if !v.Found {
			return
		}
		require.Equalf(t, float64(0), v.Val, "%s should be 0, but got %.0f", label, v.Val)
	}

	// Compliance relay assertions.
	requirePositive(comp, cq[0], "compliance relay connections_accepted")
	requirePositive(comp, cq[1], "compliance relay index_reports_received")
	requirePositive(comp, cq[2], "compliance relay index_reports_sent (failed=false)")
	requireZero(comp, cq[3], "compliance relay index_reports_sent (failed=true)")
	requireZero(comp, cq[4], "compliance relay vsock CID mismatches")
	requirePositive(comp, cq[5], "compliance relay acks_received")

	// Sensor assertions.
	requirePositive(sen, sq[0], "sensor index_reports_received")
	requirePositive(sen, sq[1], "sensor index_reports_sent (success)")
	requireZero(sen, sq[2], "sensor index_reports_sent (error)")
	requireZero(sen, sq[3], "sensor index_reports_sent (central not ready)")
	requirePositive(sen, sq[4], "sensor index_report_acks_received (ACK)")
	requireZero(sen, sq[5], "sensor enqueue_blocked")
}
