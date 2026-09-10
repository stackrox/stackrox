package scannerdefinitions

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

const (
	repoCPEMappingFetchSuccess   = "success"
	repoCPEMappingFetchUnchanged = "unchanged"
	repoCPEMappingFetchError     = "error"
)

// repoCPEMappingHashChanges counts Sensor replacing its cached mapping when
// Central's file hashes differently, including the first populate. It is not
// a per-VM VSOCK sync (see vsock_pull_sync_total).
var repoCPEMappingHashChanges = prometheus.NewCounter(prometheus.CounterOpts{
	Namespace: metrics.PrometheusNamespace,
	Subsystem: metrics.SensorSubsystem.String(),
	Name:      "repo_cpe_mapping_hash_changes_total",
	Help:      "Times Sensor replaced its cached repo-to-CPE mapping because the content hash from Central (and its upstream source) has changed. This answers the question 'How often does the repo-to-CPE mapping get updated in general?'",
})

// repoCPEMappingFetch counts each Central fetch attempt once: success is a
// hash change, unchanged is HTTP 304 or same-hash 200, error is anything that
// keeps last-good. It is not a per-VM VSOCK sync.
var repoCPEMappingFetch = prometheus.NewCounterVec(prometheus.CounterOpts{
	Namespace: metrics.PrometheusNamespace,
	Subsystem: metrics.SensorSubsystem.String(),
	Name:      "repo_cpe_mapping_fetch_total",
	Help:      "Central repo-to-CPE mapping fetches by result (success: content hash changed; unchanged: HTTP 304 or same-hash 200; error: request, validation, or unexpected status). Not a per-VM VSOCK sync.",
}, []string{"result"})

// repo2CPELastSuccessUnix is the Unix seconds of cache.lastSuccess, or 0
// until the first successful fetch. The GaugeFunc reads this so scrapes stay
// correct without a Sensor tick.
var repo2CPELastSuccessUnix atomic.Int64

func repo2CPELastSuccessSeconds() float64 {
	return float64(repo2CPELastSuccessUnix.Load())
}

// repoCPEMappingLastSuccess is Unix seconds of Sensor's last successful
// mapping fetch (304 and same-hash 200 count). Zero until the first success.
var repoCPEMappingLastSuccess = prometheus.NewGaugeFunc(prometheus.GaugeOpts{
	Namespace: metrics.PrometheusNamespace,
	Subsystem: metrics.SensorSubsystem.String(),
	Name:      "repo_cpe_mapping_last_success_timestamp_seconds",
	Help:      "Unix timestamp of Sensor's last successful repo-to-CPE mapping fetch from Central, including HTTP 304 and same-hash 200. Zero until the first success.",
}, repo2CPELastSuccessSeconds)

func init() {
	metrics.EmplaceCollector(repoCPEMappingHashChanges, repoCPEMappingFetch, repoCPEMappingLastSuccess)
}
