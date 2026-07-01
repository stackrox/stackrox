package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

// StartTimeToMS allows to record sub-millisecond durations. Without this, things faster than 1ms are rounded to 0.
func StartTimeToMS(t time.Time) float64 {
	return float64(time.Since(t).Nanoseconds()) / float64(time.Millisecond)
}

var (
	StatusCentralNotReadyLabels = prometheus.Labels{"status": "central not ready"}
	StatusErrorLabels           = prometheus.Labels{"status": "error"}
	StatusSuccessLabels         = prometheus.Labels{"status": "success"}
	StatusTimeoutLabels         = prometheus.Labels{"status": "timeout"}
)

// IndexReportsReceived is a counter for the number of virtual machine index reports received.
// Asserted in VM E2E tests (tests/vm_scanning_metrics_test.go). Update tests when renaming or removing.
var IndexReportsReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_reports_received_total",
		Help:      "Total number of virtual machine index reports received by this Sensor",
	},
)

// IndexReportsSent is a counter for the number of virtual machine index reports sent.
// Asserted in VM E2E tests (tests/vm_scanning_metrics_test.go). Update tests when renaming or removing.
var IndexReportsSent = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_reports_sent_total",
		Help:      "Total number of virtual machine index reports sent by this Sensor",
	},
	[]string{"status"},
)

// VirtualMachineIndexReportHandlingDurationMilliseconds captures how long it takes to handle a virtual machine index report.
var VirtualMachineIndexReportHandlingDurationMilliseconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_report_handling_duration_milliseconds",
		Help:      "Distribution of time spent (in ms) handling virtual machine index reports in Sensor, including the enqueue step",
		Buckets:   prometheus.ExponentialBuckets(10, 2, 12), // 10ms to ~40s
	},
)

// IndexReportProcessingDuration label values.
const (
	// IndexReportHandlingMessageToCentralSuccess marks processing flows that successfully send to Central.
	IndexReportHandlingMessageToCentralSuccess = "success"
	// IndexReportHandlingMessageToCentralNilReport marks flows that exit because the report was nil.
	IndexReportHandlingMessageToCentralNilReport = "nil_report"
	// IndexReportHandlingMessageToCentralInvalidCID marks flows that exit because the message could not be constructed due to an invalid vsock CID.
	IndexReportHandlingMessageToCentralInvalidCID = "invalid_vsock_cid"
	// IndexReportHandlingMessageToCentralVMUnknown marks flows that exit because the virtual machine is not known to Sensor.
	IndexReportHandlingMessageToCentralVMUnknown = "vm_unknown_to_sensor"
)

// IndexReportProcessingDurationMilliseconds tracks how long Sensor spends processing index reports after dequeuing them.
var IndexReportProcessingDurationMilliseconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_report_processing_duration_milliseconds",
		Help:      "Distribution of time spent (in ms) processing virtual machine index reports after reading from indexReports and before sending to Central",
		Buckets:   prometheus.ExponentialBuckets(10, 2, 12),
	},
	[]string{"outcome"},
)

// IndexReportEnqueueOutcome label values for enqueue latency observations.
const (
	IndexReportEnqueueOutcomeSuccess  = "success"
	IndexReportEnqueueOutcomeTimeout  = "context_timeout"
	IndexReportEnqueueOutcomeCanceled = "context_canceled"
)

// IndexReportBlockingEnqueueDurationMilliseconds measures how long Sensor waits after detecting backpressure.
var IndexReportBlockingEnqueueDurationMilliseconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_report_blocking_enqueue_duration_milliseconds",
		Help:      "Time spent (in ms) waiting for indexReports capacity after encountering a full channel",
		Buckets:   append([]float64{1, 5, 10, 50, 100, 250, 500}, prometheus.ExponentialBuckets(1000, 2, 8)...), // 1ms to 128s
	},
	[]string{"outcome"},
)

// IndexReportEnqueueBlockedTotal counts how often the enqueue channel was full.
// Asserted in VM E2E tests (tests/vm_scanning_metrics_test.go). Update tests when renaming or removing.
var IndexReportEnqueueBlockedTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_report_enqueue_blocked_total",
		Help:      "Number of times virtual machine index report enqueue attempts found the indexReports channel full",
	},
)

// VMDiscoveredData is a counter for VM discovered data grouped by detected OS and status values.
var VMDiscoveredData = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_discovered_data_total",
		Help:      "Total number of VM index reports received by Sensor grouped by detected OS and discovered data status values",
	},
	[]string{"detected_os", "activation_status", "dnf_metadata_status"},
)

// VMDiscoveredDataDNFStatus is a counter for individual DNF status flags observed
// on VMs, reported by either push- or pull-mode roxagent. This avoids high-cardinality
// label combinations by tracking one flag per sample.
var VMDiscoveredDataDNFStatus = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_discovered_data_dnf_status_total",
		Help:      "Total number of DNF status flags observed in VM index reports received by Sensor",
	},
	[]string{"dnf_status"},
)

// IndexReportAcksReceived counts ACK/NACK responses received from Central for VM index reports.
// Asserted in VM E2E tests (tests/vm_scanning_metrics_test.go). Update tests when renaming or removing.
var IndexReportAcksReceived = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_report_acks_received_total",
		Help:      "Total number of ACK/NACK responses received from Central for VM index reports",
	},
	[]string{"action"}, // "ACK" or "NACK"
)

// Pull-mode request status label values for PullRequestsTotal.
const (
	PullStatusSuccess       = "success"
	PullStatusUnchanged     = "unchanged"
	PullStatusDialError     = "dial_error"
	PullStatusReadError     = "read_error"
	PullStatusInvalidReport = "invalid_report"
	PullStatusSendError     = "send_error"
	PullStatusNotReady      = "not_ready"
	PullStatusUnknownMethod = "unknown_method"
	PullStatusTimeout       = "timeout"
)

// PullDialDurationSeconds measures time to establish a websocket connection per VM.
var PullDialDurationSeconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_dial_duration_seconds",
		Help:      "Time to establish websocket connection to a VM agent",
		Buckets:   prometheus.ExponentialBuckets(0.01, 2, 12), // 10ms to ~20s
	},
)

// PullReadDurationSeconds measures time to receive the full response from a VM agent.
var PullReadDurationSeconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_read_duration_seconds",
		Help:      "Time to receive full response from a VM agent",
		Buckets:   prometheus.ExponentialBuckets(0.05, 2, 11), // 50ms to ~51s
	},
)

// PullTotalDurationSeconds measures end-to-end time per VM (dial + read + send to Central).
var PullTotalDurationSeconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_total_duration_seconds",
		Help:      "End-to-end duration per VM: dial + read + send to Central",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 11), // 100ms to ~102s
	},
)

// PullCycleDurationSeconds measures the full poll cycle across all VMs.
var PullCycleDurationSeconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_cycle_duration_seconds",
		Help:      "Duration of a full poll cycle across all VMs",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~512s
	},
)

// PullReportBytes measures response payload size in bytes.
var PullReportBytes = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_report_bytes",
		Help:      "Response payload size in bytes from VM agent",
		Buckets:   prometheus.ExponentialBuckets(1024, 2, 14), // 1KB to ~8MB
	},
)

// PullReportPackages measures package count per report.
var PullReportPackages = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_report_packages",
		Help:      "Number of packages per VM index report",
		Buckets:   prometheus.ExponentialBuckets(10, 2, 10), // 10 to ~5120
	},
)

// PullRequestsTotal counts per-VM pull attempts by status.
var PullRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_requests_total",
		Help:      "Per-VM pull attempts by outcome status",
	},
	[]string{"status"},
)

// PullCyclesTotal counts poll cycles executed.
var PullCyclesTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_cycles_total",
		Help:      "Total number of pull poll cycles executed",
	},
)

// PullVMsInCycle tracks the number of running VMs in the last poll set.
var PullVMsInCycle = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_vms_in_cycle",
		Help:      "Number of running VMs in the last poll set",
	},
)

func init() {
	prometheus.MustRegister(
		// Push-mode metrics.
		IndexReportsReceived,
		IndexReportsSent,
		VirtualMachineIndexReportHandlingDurationMilliseconds,
		IndexReportProcessingDurationMilliseconds,
		IndexReportBlockingEnqueueDurationMilliseconds,
		IndexReportEnqueueBlockedTotal,
		VMDiscoveredData,
		VMDiscoveredDataDNFStatus,
		IndexReportAcksReceived,
		// Pull-mode metrics.
		PullDialDurationSeconds,
		PullReadDurationSeconds,
		PullTotalDurationSeconds,
		PullCycleDurationSeconds,
		PullReportBytes,
		PullReportPackages,
		PullRequestsTotal,
		PullCyclesTotal,
		PullVMsInCycle,
	)
}
