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

// IndexReportsSent is a counter for the number of virtual machine index reports sent.
var IndexReportsSent = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_reports_sent_total",
		Help:      "Total number of virtual machine index reports sent by this Sensor",
	},
	[]string{"status"},
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
var IndexReportEnqueueBlockedTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_report_enqueue_blocked_total",
		Help:      "Number of times virtual machine index report enqueue attempts found the indexReports channel full",
	},
)

// VMDiscoveredData counts VM discovered-data observations grouped by detected OS and status values.
var VMDiscoveredData = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_discovered_data_total",
		Help:      "Total number of VM index reports received by Sensor grouped by detected OS and discovered data status values",
	},
	[]string{"detected_os", "activation_status", "dnf_metadata_status"},
)

// IndexReportAcksReceived counts ACK/NACK responses received from Central for VM index reports.
var IndexReportAcksReceived = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_report_acks_received_total",
		Help:      "Total number of ACK/NACK responses received from Central for VM index reports",
	},
	[]string{"action"}, // "ACK" or "NACK"
)

// Pull-mode outcome labels are split across three counters so transport,
// GetReport protocol, and scrape-pipeline results stay distinct as more
// VSOCK RPC methods are added later. Each per-VM pull attempt increments
// exactly one of these counters (mutually exclusive partition).

// Transport-layer status values for PullTransportTotal.
const (
	PullTransportDialError     = "dial_error"
	PullTransportTimeout       = "timeout"
	PullTransportReadError     = "read_error"
	PullTransportAbnormalClose = "abnormal_close"
	PullTransportUnexpected    = "unexpected"
)

// GetReport protocol status values for PullGetReportTotal.
const (
	PullGetReportUnchanged         = "unchanged"
	PullGetReportNotReady          = "not_ready"
	PullGetReportMappingRequired   = "mapping_required"
	PullGetReportUnknownMethod     = "unknown_method"
	PullGetReportBusy              = "busy"
	PullGetReportInternalError     = "internal_error"
	PullGetReportMalformedRequest  = "malformed_request"
	PullGetReportRequestTooLarge   = "request_too_large"
	PullGetReportUnknownAgentError = "unknown_agent_error"
)

// Scrape-pipeline status values for PullScrapeTotal (post-GetReport).
const (
	PullScrapeSuccess       = "success"
	PullScrapeInvalidReport = "invalid_report"
	PullScrapeSendError     = "send_error"
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

// PullTickDurationSeconds measures how long each scraper tick spends
// scraping the VMs that were due, not a poll of the whole VM set: VMs are
// scraped on independent per-VM schedules, not in lockstep.
var PullTickDurationSeconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_tick_duration_seconds",
		Help:      "Duration of a scraper tick spent scraping the VMs due at that tick",
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
		// 1KB to ~32MB brackets the 16MB default response-size ceiling
		// (env.VirtualMachinesPullMaxResponseSizeKB) with a bucket to spare,
		// giving the >8MB range that reportcheck.IsViable flags as
		// "unusually large" its own resolution up to that ceiling.
		Buckets: prometheus.ExponentialBuckets(1024, 2, 16),
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

// PullTransportTotal counts per-VM pull attempts that failed at the VSOCK
// transport layer (dial / read / abnormal close) before a protocol result.
var PullTransportTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_transport_total",
		Help:      "Per-VM pull attempts that failed at the VSOCK transport layer",
	},
	[]string{"status"},
)

// PullGetReportTotal counts per-VM GetReport protocol outcomes (Unchanged,
// ErrorCode sentinels). Transport failures are counted on PullTransportTotal
// instead; successful full reports continue into the scrape pipeline and are
// counted on PullScrapeTotal.
var PullGetReportTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_get_report_total",
		Help:      "Per-VM GetReport protocol outcomes (unchanged and agent ErrorCode results)",
	},
	[]string{"status"},
)

// PullScrapeTotal counts per-VM scrape-pipeline outcomes after a full
// GetReport payload was received (viability check, send to Central, success).
var PullScrapeTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_scrape_total",
		Help:      "Per-VM scrape-pipeline outcomes after a full GetReport payload was received",
	},
	[]string{"status"},
)

// PullTicksTotal counts scraper ticks executed.
var PullTicksTotal = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_ticks_total",
		Help:      "Total number of scraper ticks executed",
	},
)

// PullTrackedVMs tracks the number of VMs currently tracked for pull-mode
// scraping, regardless of how many were due at the last tick.
var PullTrackedVMs = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_tracked_vms",
		Help:      "Number of VMs currently tracked for pull-mode scraping",
	},
)

// PullDueVMs is how many VMs were eligible to scrape at the start of the last
// tick (nextAttemptAt had arrived and they were not already in flight).
var PullDueVMs = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_due_vms",
		Help:      "How many VMs were eligible to scrape at the beginning of the last scraper tick",
	},
)

// PullStartsPerTick is how many VM scrapes each tick launches. Idle ticks
// (nobody due) are omitted so the histogram is not dominated by zeros.
var PullStartsPerTick = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_starts_per_tick",
		Help: "How many VM scrapes the scraper starts in a single tick. " +
			"Idle ticks are not observed. Compare with vsock_pull_due_vms: " +
			"spread due times keep both small; a mass of large starts is a dump.",
		Buckets: []float64{0, 1, 2, 3, 5, 8, 10, 15, 20, 30, 50, 100},
	},
)

// PullForwardInterarrivalSeconds is the Sensor-level gap between consecutive
// successful forwards to Central. The first forward after start is not observed.
var PullForwardInterarrivalSeconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_forward_interarrival_seconds",
		Help: "Seconds between consecutive successful VM index-report forwards " +
			"from this Sensor to Central. The first forward after Sensor start does not count.",
		// 10ms to ~47h. Sized for a 24h poll in extreme cases so those
		// gaps stay in a finite bucket instead of +Inf.
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 25),
	},
)

// PullScheduleOffsetSeconds is the random extra delay drawn when a VM returns
// to cadence after success or a permanent non-retry outcome (retries/NACKs do not).
var PullScheduleOffsetSeconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "vsock_pull_schedule_offset_seconds",
		Help: "Random extra delay (seconds) added on top of the poll interval " +
			"when scheduling a VM's next attempt after a return-to-cadence outcome.",
		// 250ms to ~36h. Sized for a 24h poll in extreme cases (W up to 24h).
		Buckets: prometheus.ExponentialBuckets(0.25, 2, 20),
	},
)

func init() {
	prometheus.MustRegister(
		IndexReportsSent,
		IndexReportProcessingDurationMilliseconds,
		IndexReportBlockingEnqueueDurationMilliseconds,
		IndexReportEnqueueBlockedTotal,
		VMDiscoveredData,
		IndexReportAcksReceived,
		PullDialDurationSeconds,
		PullReadDurationSeconds,
		PullTotalDurationSeconds,
		PullTickDurationSeconds,
		PullReportBytes,
		PullReportPackages,
		PullTransportTotal,
		PullGetReportTotal,
		PullScrapeTotal,
		PullTicksTotal,
		PullTrackedVMs,
		PullDueVMs,
		PullStartsPerTick,
		PullForwardInterarrivalSeconds,
		PullScheduleOffsetSeconds,
	)
}
