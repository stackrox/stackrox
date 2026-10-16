package matcher

import "github.com/prometheus/client_golang/prometheus"

// vulnerabilityBundleInfo is the planned Matcher-local identity collector.
// Registration and updates are intentionally deferred to Task 2 implementation.
var vulnerabilityBundleInfo = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Name: "rox_scanner_vulnerability_bundle_info",
	Help: "SHA-256 of the last successfully imported complete vulnerability ZIP.",
}, []string{"sha256"})

// vulnerabilityBundleImportErrors counts failed attempts to import an internal
// vulnerability bundle. Skipped and already-current bundles are not errors.
// Registration and incrementing are deferred to the implementation phase.
var vulnerabilityBundleImportErrors = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "rox_scanner_vulnerability_bundle_import_errors_total",
	Help: "Total failed vulnerability bundle import attempts in Scanner V4 Matcher.",
})
