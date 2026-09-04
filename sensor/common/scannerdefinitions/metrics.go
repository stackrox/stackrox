package scannerdefinitions

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
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

func init() {
	metrics.EmplaceCollector(repoCPEMappingHashChanges)
}
