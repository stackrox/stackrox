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
var IndexReportsReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_reports_received_total",
		Help:      "Total number of virtual machine index reports received by this Sensor",
	},
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

func init() {
	prometheus.MustRegister(
		IndexReportsReceived,
		IndexReportsSent,
		VirtualMachineIndexReportHandlingDurationMilliseconds,
		IndexReportProcessingDurationMilliseconds,
		IndexReportBlockingEnqueueDurationMilliseconds,
		IndexReportEnqueueBlockedTotal,
		VMDiscoveredData,
	)
}
