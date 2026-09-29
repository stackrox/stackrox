package vmscraper

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/sensor/common/virtualmachine/metrics"
)

// logAndRecordDiscoveredFacts logs and records metrics for the VM facts
// (detected OS, activation status, DNF metadata status) that roxagent reports
// alongside every pulled index report.
func logAndRecordDiscoveredFacts(key string, facts map[string]string) {
	if len(facts) == 0 {
		return
	}

	detectedOS := facts["detected_os"]
	osVersion := facts["os_version"]
	activationStatus := facts["activation_status"]
	dnfMetadataStatus := facts["dnf_metadata_status"]

	log.Debugf("VMScraper: VM discovered data for %q: detected_os=%s, os_version=%q, activation_status=%s, dnf_metadata_status=%s",
		key, detectedOS, osVersion, activationStatus, dnfMetadataStatus)

	metrics.VMDiscoveredData.With(prometheus.Labels{
		"detected_os":         detectedOS,
		"activation_status":   activationStatus,
		"dnf_metadata_status": dnfMetadataStatus,
	}).Inc()
}
