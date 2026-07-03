package vmscraper

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
)

// scanTriggerFactKey is the ResponseMeta.facts key roxagent sets to classify
// why a report was generated.
const scanTriggerFactKey = "scan_trigger"

// scanTriggerReactive is the scanTriggerFactKey value for reports produced by
// a reactive (event-triggered) rescan, as opposed to a routine scheduled one.
const scanTriggerReactive = "reactive"

// scanTriggerScheduled is the scanTriggerFactKey value for reports produced
// by a routine, non-reactive scan.
const scanTriggerScheduled = "scheduled"

// isReactiveTrigger reports whether facts indicate a reactive rescan. Absent
// or unrecognized values default to false (scheduled), the safe default for
// older roxagent versions that don't set this fact.
func isReactiveTrigger(facts map[string]string) bool {
	return facts[scanTriggerFactKey] == scanTriggerReactive
}

// logAndRecordDiscoveredFacts logs and records metrics for the VM facts
// (detected OS, activation status, DNF status, scan trigger) that roxagent
// reports alongside every pulled index report. This mirrors the logging/metrics
// that push-mode agents produce via UpsertVirtualMachineIndexReport, so
// operators see the same "VM discovered data" signal regardless of which
// transport mode delivered it.
func logAndRecordDiscoveredFacts(key string, facts map[string]string) {
	if len(facts) == 0 {
		return
	}

	detectedOS := facts["detected_os"]
	osVersion := facts["os_version"]
	activationStatus := facts["activation_status"]
	dnfMetadataStatus := facts["dnf_metadata_status"]
	dnfStatus := facts["dnf_status"]
	scanTrigger := facts[scanTriggerFactKey]

	log.Debugf("VMScraper: VM discovered data for %q: detected_os=%s, os_version=%q, activation_status=%s, dnf_status=[%s], scan_trigger=%q",
		key, detectedOS, osVersion, activationStatus, dnfStatus, scanTrigger)

	metrics.VMDiscoveredData.With(prometheus.Labels{
		"detected_os":         detectedOS,
		"activation_status":   activationStatus,
		"dnf_metadata_status": dnfMetadataStatus,
	}).Inc()
	recordDnfStatusMetrics(dnfStatus)
	recordScanTriggerMetric(scanTrigger)
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
		metrics.VMDiscoveredDataDNFStatus.WithLabelValues(name).Inc()
	}
}

// recordScanTriggerMetric records the scan_trigger classification, whitelisting
// known values and falling back to "unknown" for anything else (older
// roxagent versions that don't set the fact yet, typos, or future trigger
// types). roxagent-supplied data crosses a trust boundary, so this bounds the
// metric's label cardinality rather than passing arbitrary values through.
func recordScanTriggerMetric(scanTrigger string) {
	if scanTrigger != scanTriggerReactive && scanTrigger != scanTriggerScheduled {
		scanTrigger = "unknown"
	}
	metrics.VMDiscoveredDataScanTrigger.WithLabelValues(scanTrigger).Inc()
}
