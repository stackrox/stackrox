package manager

import (
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/timestamp"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
)

func benchmarkConnStatus(now timestamp.MicroTS) *connStatus {
	return &connStatus{
		firstSeen:             now.Add(-env.ContainerIDResolutionGracePeriod.DurationSetting() - time.Second),
		lastSeen:              now,
		containerIDFound:      true,
		historicalContainerID: true,
		rotten:                true,
	}
}

func BenchmarkUpdateEndpointMetric(b *testing.B) {
	flowMetrics.FlowEnrichmentEventsEndpoint.Reset()
	flowMetrics.HostProcessesEnrichmentEvents.Reset()

	now := timestamp.Now()
	action := PostEnrichmentActionCheckRemove
	resultNG := EnrichmentResultSuccess
	resultPLOP := EnrichmentResultSkipped
	reasonNG := EnrichmentReasonEpSuccessInactive
	reasonPLOP := EnrichmentReasonEpFeaturePlopDisabled
	status := benchmarkConnStatus(now)
	updateEndpointMetric(now, action, resultNG, resultPLOP, reasonNG, reasonPLOP, status)

	b.ReportAllocs()
	for b.Loop() {
		updateEndpointMetric(now, action, resultNG, resultPLOP, reasonNG, reasonPLOP, status)
	}
}

func BenchmarkUpdateConnectionMetric(b *testing.B) {
	flowMetrics.FlowEnrichmentEventsConnection.Reset()

	now := timestamp.Now()
	action := PostEnrichmentActionCheckRemove
	result := EnrichmentResultSuccess
	reason := EnrichmentReasonConnSuccess
	status := benchmarkConnStatus(now)
	status.isExternal = true
	updateConnectionMetric(now, action, result, reason, status)

	b.ReportAllocs()
	for b.Loop() {
		updateConnectionMetric(now, action, result, reason, status)
	}
}
