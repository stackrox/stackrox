package vmscraper

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
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
	for _, name := range strings.Split(dnfStatus, ", ") {
		if name == "" {
			continue
		}
		metrics.VMDiscoveredDataDNFStatus.WithLabelValues(name).Inc()
	}
}
