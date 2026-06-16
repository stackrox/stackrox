//go:build test_e2e_vm

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/tests/testmetrics"
	"github.com/stackrox/rox/tests/vmhelpers"
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

// assertPipelineMetrics scrapes compliance and sensor metrics, retrying until
// all pipeline assertions pass or the timeout expires.
// vmNodeName must be non-empty so compliance metrics are scoped to the VM's local collector pod.
func (s *VMScanningSuite) assertPipelineMetrics(ctx context.Context, t require.TestingT, vmNodeName string) {
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
		comp, err := testmetrics.ScrapeComponent(ctx, s.k8sClient, compTarget)
		if !assert.NoError(ct, err, "scrape compliance") {
			return
		}
		sen, err := testmetrics.ScrapeComponent(ctx, s.k8sClient, senTarget)
		if !assert.NoError(ct, err, "scrape sensor") {
			return
		}
		assertPipeline(ct, comp, sen)
	}, metricsTimeout, metricsPoll)
}

func assertPipeline(t *assert.CollectT, comp, sen testmetrics.Metrics) {
	positive := func(m testmetrics.Metrics, name string, labels ...string) float64 {
		val, found := m.GetValue(name, labels...)
		assert.Truef(t, found, "%s should be present", name)
		assert.Greaterf(t, val, float64(0), "%s should be > 0", name)
		return val
	}

	zero := func(m testmetrics.Metrics, name string, labels ...string) {
		val, found := m.GetValue(name, labels...)
		if !found {
			return
		}
		assert.Equalf(t, float64(0), val, "%s should be 0", name)
	}

	// Compliance relay: full receive → send → ack cycle.
	compReceived := positive(comp, vmhelpers.MetricComplianceRelayIndexReportsReceivedTotal)
	compSentOK := positive(comp, vmhelpers.MetricComplianceRelayIndexReportsSentTotal, "failed", "false")
	positive(comp, vmhelpers.MetricComplianceRelayConnectionsAcceptedTotal)
	positive(comp, vmhelpers.MetricComplianceRelayIndexReportAcksReceivedTotal)
	zero(comp, vmhelpers.MetricComplianceRelayIndexReportsSentTotal, "failed", "true")
	zero(comp, vmhelpers.MetricComplianceRelayIndexReportsMismatchingVsockTotal)

	// Sensor: full receive → send → ack cycle.
	senReceived := positive(sen, vmhelpers.MetricSensorVMIndexReportsReceivedTotal)
	senSentOK := positive(sen, vmhelpers.MetricSensorVMIndexReportsSentTotal, "status", vmhelpers.SensorIndexReportStatusSuccess)
	positive(sen, vmhelpers.MetricSensorVMIndexReportAcksReceivedTotal, "action", "ACK")
	zero(sen, vmhelpers.MetricSensorVMIndexReportsSentTotal, "status", vmhelpers.SensorIndexReportStatusError)
	zero(sen, vmhelpers.MetricSensorVMIndexReportsSentTotal, "status", vmhelpers.SensorIndexReportStatusCentralNotReady)
	zero(sen, vmhelpers.MetricSensorVMIndexReportEnqueueBlockedTotal)

	// Relational invariants: can't send more than received.
	assert.GreaterOrEqualf(t, compReceived, compSentOK,
		"compliance: received (%.0f) should be >= sent_ok (%.0f)", compReceived, compSentOK)
	assert.GreaterOrEqualf(t, senReceived, senSentOK,
		"sensor: received (%.0f) should be >= sent_ok (%.0f)", senReceived, senSentOK)
}
