//go:build test_e2e_vm

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/tests/testmetrics"
	"github.com/stretchr/testify/assert"
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

const (
	metricsTimeout = 2 * time.Minute
	metricsPoll    = 10 * time.Second
)

type pipelineMetricsSnapshot struct {
	compliance testmetrics.Metrics
	sensor     testmetrics.Metrics
}

// assertPipelineMetrics scrapes compliance and sensor metrics, retrying until
// all pipeline assertions pass or the timeout expires.
// vmNodeName must be non-empty so compliance metrics are scoped to the VM's local collector pod.
func (s *VMScanningSuite) assertPipelineMetrics(ctx context.Context, t require.TestingT, vmNodeName string, before, midpoint pipelineMetricsSnapshot) {
	tt, ok := t.(*testing.T)
	require.True(t, ok, "assertPipelineMetrics requires *testing.T")

	require.NotEmpty(t, vmNodeName,
		"VM node name must be known before asserting pipeline metrics; "+
			"cluster-wide collector scraping is not supported because it conflates metrics from unrelated VMs")

	compTarget := s.complianceTarget(vmNodeName)
	s.logf("pipeline metrics: VM node=%q, compliance selector=%q field=%q",
		vmNodeName, compTarget.LabelSelector, compTarget.FieldSelector)

	senTarget := s.sensorTarget()

	assert.EventuallyWithT(tt, func(ct *assert.CollectT) {
		after, err := s.scrapePipelineMetrics(ctx, compTarget, senTarget)
		if !assert.NoError(ct, err, "scrape pipeline metrics") {
			return
		}
		assertPipeline(ct, before, midpoint, after)
	}, metricsTimeout, metricsPoll)
}

func (s *VMScanningSuite) mustScrapePipelineMetrics(ctx context.Context, t require.TestingT, vmNodeName string) pipelineMetricsSnapshot {
	require.NotEmpty(t, vmNodeName,
		"VM node name must be known before asserting pipeline metrics; "+
			"cluster-wide collector scraping is not supported because it conflates metrics from unrelated VMs")

	compTarget := s.complianceTarget(vmNodeName)
	s.logf("pipeline metrics: VM node=%q, compliance selector=%q field=%q",
		vmNodeName, compTarget.LabelSelector, compTarget.FieldSelector)

	senTarget := s.sensorTarget()
	snapshot, err := s.scrapePipelineMetrics(ctx, compTarget, senTarget)
	require.NoError(t, err, "scrape pipeline metrics baseline")
	return snapshot
}

func (s *VMScanningSuite) scrapePipelineMetrics(ctx context.Context, compTarget, senTarget testmetrics.ScrapeTarget) (pipelineMetricsSnapshot, error) {
	comp, err := testmetrics.ScrapeComponent(ctx, s.k8sClient, compTarget)
	if err != nil {
		return pipelineMetricsSnapshot{}, err
	}
	sen, err := testmetrics.ScrapeComponent(ctx, s.k8sClient, senTarget)
	if err != nil {
		return pipelineMetricsSnapshot{}, err
	}
	return pipelineMetricsSnapshot{
		compliance: comp,
		sensor:     sen,
	}, nil
}

type metricObservation struct {
	value   float64
	present bool
}

func metricObservationFor(m testmetrics.Metrics, name string, labels ...string) metricObservation {
	val, found := m.GetValue(name, labels...)
	return metricObservation{
		value:   val,
		present: found,
	}
}

func (o metricObservation) valueOrZero() float64 {
	if !o.present {
		return 0
	}
	return o.value
}

func formatMetricObservation(o metricObservation) string {
	if !o.present {
		return "<absent>"
	}
	return fmt.Sprintf("%.0f", o.value)
}

func assertMetricDeltaAtLeast(t *assert.CollectT, before, after testmetrics.Metrics, wantMin float64, name string, labels ...string) {
	beforeObs := metricObservationFor(before, name, labels...)
	afterObs := metricObservationFor(after, name, labels...)
	got := afterObs.valueOrZero() - beforeObs.valueOrZero()
	assert.Truef(t, afterObs.present,
		"%s should be present after the scenario (before=%s after=%s delta=%.0f)",
		name, formatMetricObservation(beforeObs), formatMetricObservation(afterObs), got)
	assert.GreaterOrEqualf(t, got, wantMin, "%s delta should be >= %.0f (before=%s after=%s delta=%.0f)",
		name, wantMin, formatMetricObservation(beforeObs), formatMetricObservation(afterObs), got)
}

func assertMetricZeroOrAbsentDelta(t *assert.CollectT, before, after testmetrics.Metrics, name string, labels ...string) {
	beforeObs := metricObservationFor(before, name, labels...)
	afterObs := metricObservationFor(after, name, labels...)
	got := afterObs.valueOrZero() - beforeObs.valueOrZero()
	assert.Equalf(t, float64(0), got, "%s delta should be 0 or absent (before=%s after=%s delta=%.0f)",
		name, formatMetricObservation(beforeObs), formatMetricObservation(afterObs), got)
}

func metricDelta(before, after testmetrics.Metrics, name string, labels ...string) float64 {
	afterObs := metricObservationFor(after, name, labels...)
	beforeObs := metricObservationFor(before, name, labels...)
	return afterObs.valueOrZero() - beforeObs.valueOrZero()
}

type metricPhaseDeltas struct {
	firstRun float64
	rescan   float64
	total    float64
}

func metricPhaseDeltasFor(before, midpoint, after testmetrics.Metrics, name string, labels ...string) metricPhaseDeltas {
	return metricPhaseDeltas{
		firstRun: metricDelta(before, midpoint, name, labels...),
		rescan:   metricDelta(midpoint, after, name, labels...),
		total:    metricDelta(before, after, name, labels...),
	}
}

// assertPipeline verifies the two-report scenario using three metric snapshots:
// baseline before the first roxagent run, midpoint after the first report is
// visible in Central and just before rescan, and final after the rescan path
// completes. The main contract is that two ACKs make it all the way back to
// compliance; the other metrics help localize failures when that contract is not met.
func assertPipeline(t *assert.CollectT, before, midpoint, after pipelineMetricsSnapshot) {
	complianceBefore := before.compliance
	complianceMidpoint := midpoint.compliance
	complianceAfter := after.compliance
	sensorBefore := before.sensor
	sensorMidpoint := midpoint.sensor
	sensorAfter := after.sensor

	complianceACKMetric := "rox_compliance_virtual_machine_index_acks_from_sensor_total"
	sensorACKMetric := "rox_sensor_virtual_machine_index_report_acks_received_total"
	complianceIngressMetric := "rox_compliance_virtual_machine_relay_index_reports_received_total"

	complianceACKDeltas := metricPhaseDeltasFor(complianceBefore, complianceMidpoint, complianceAfter, complianceACKMetric, "action", "ACK")
	sensorACKDeltas := metricPhaseDeltasFor(sensorBefore, sensorMidpoint, sensorAfter, sensorACKMetric, "action", "ACK")
	complianceIngressDeltas := metricPhaseDeltasFor(complianceBefore, complianceMidpoint, complianceAfter, complianceIngressMetric)

	// Primary end-to-end assertion: the two roxagent runs in this test should
	// yield exactly two ACKs returning from Sensor to compliance.
	assert.Equalf(t, float64(2), complianceACKDeltas.total,
		"%s delta should be 2; compliance ACKs firstRun=%.0f rescan=%.0f total=%.0f; "+
			"sensor ACKs firstRun=%.0f rescan=%.0f total=%.0f; "+
			"compliance relay ingress firstRun=%.0f rescan=%.0f total=%.0f",
		complianceACKMetric,
		complianceACKDeltas.firstRun, complianceACKDeltas.rescan, complianceACKDeltas.total,
		sensorACKDeltas.firstRun, sensorACKDeltas.rescan, sensorACKDeltas.total,
		complianceIngressDeltas.firstRun, complianceIngressDeltas.rescan, complianceIngressDeltas.total)

	// Breadcrumbs: use "at least 2" because this test sends two reports, so a
	// healthy path should move these counters by at least two, but they are also
	// shared cumulative metrics that may legitimately observe extra activity
	// (for example retries or other VM traffic). Requiring equality here would
	// make the diagnostics more flaky than the primary compliance ACK check.
	assertMetricDeltaAtLeast(t, sensorBefore, sensorAfter, 2,
		sensorACKMetric, "action", "ACK")
	assertMetricDeltaAtLeast(t, complianceBefore, complianceAfter, 2,
		complianceIngressMetric)

	// Happy-path guardrails: none of the known error or backpressure counters
	// should move during the two-report scenario.
	assertMetricZeroOrAbsentDelta(t, complianceBefore, complianceAfter,
		"rox_compliance_virtual_machine_index_acks_from_sensor_total", "action", "NACK")
	assertMetricZeroOrAbsentDelta(t, complianceBefore, complianceAfter,
		"rox_compliance_virtual_machine_relay_index_reports_sent_total", "failed", "true")
	assertMetricZeroOrAbsentDelta(t, complianceBefore, complianceAfter,
		"rox_compliance_virtual_machine_relay_index_reports_mismatching_vsock_cid_total")
	assertMetricZeroOrAbsentDelta(t, sensorBefore, sensorAfter,
		"rox_sensor_virtual_machine_index_report_acks_received_total", "action", "NACK")
	assertMetricZeroOrAbsentDelta(t, sensorBefore, sensorAfter,
		"rox_sensor_virtual_machine_index_reports_sent_total", "status", "error")
	assertMetricZeroOrAbsentDelta(t, sensorBefore, sensorAfter,
		"rox_sensor_virtual_machine_index_reports_sent_total", "status", "central not ready")
	assertMetricZeroOrAbsentDelta(t, sensorBefore, sensorAfter,
		"rox_sensor_virtual_machine_index_report_enqueue_blocked_total")
}
