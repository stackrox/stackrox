package vmscraper

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
)

// knownDnfStatusFlags whitelists the DnfStatusFlag names roxagent can report
// (see v1.DnfStatusFlag in internalapi/virtualmachine/v1/index_report.proto).
// Facts arrive here as a flat string rather than the typed enum, so this
// mirrors those names by hand. Any fragment outside this set - a malformed
// value, or one from a newer agent this Sensor doesn't recognize yet -
// collapses into the bounded "unknown" label instead of minting a new
// Prometheus child metric per arbitrary string.
var knownDnfStatusFlags = set.NewStringSet(
	"DNF_REPO_CONFIG_FOUND",
	"DNF_V4_CACHE_FOUND",
	"DNF_V5_CACHE_FOUND",
	"DNF_V4_HISTORY_DB_FOUND",
	"DNF_V5_HISTORY_DB_FOUND",
)

// logAndRecordDiscoveredFacts logs and records metrics for the VM facts
// (detected OS, activation status, DNF status) that roxagent reports alongside
// every pulled index report. This mirrors the logging/metrics that push-mode
// agents produce via UpsertVirtualMachineIndexReport, so operators see the same
// "VM discovered data" signal regardless of which transport mode delivered it.
func logAndRecordDiscoveredFacts(key string, facts map[string]string) {
	if len(facts) == 0 {
		return
	}

	detectedOS := facts["detected_os"]
	osVersion := facts["os_version"]
	activationStatus := facts["activation_status"]
	dnfMetadataStatus := facts["dnf_metadata_status"]
	dnfStatus := facts["dnf_status"]

	log.Debugf("VMScraper: VM discovered data for %q: detected_os=%s, os_version=%q, activation_status=%s, dnf_status=[%s]",
		key, detectedOS, osVersion, activationStatus, dnfStatus)

	metrics.VMDiscoveredData.With(prometheus.Labels{
		"detected_os":         detectedOS,
		"activation_status":   activationStatus,
		"dnf_metadata_status": dnfMetadataStatus,
	}).Inc()
	recordDnfStatusMetrics(dnfStatus)
}

// recordDnfStatusMetrics splits the comma-joined "name1, name2" DNF status
// string (as produced by roxagent's formatDnfStatusFlags) and increments the
// shared low-cardinality counter once per flag.
func recordDnfStatusMetrics(dnfStatus string) {
	if dnfStatus == "" || dnfStatus == "none" {
		metrics.VMDiscoveredDataDNFStatus.WithLabelValues("none").Inc()
		return
	}
	for name := range strings.SplitSeq(dnfStatus, ", ") {
		if name == "" {
			continue
		}
		if !knownDnfStatusFlags.Contains(name) {
			name = "unknown"
		}
		metrics.VMDiscoveredDataDNFStatus.WithLabelValues(name).Inc()
	}
}
