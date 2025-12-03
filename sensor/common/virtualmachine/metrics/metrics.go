package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

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
	// IndexReportProcessingOutcomeSuccess marks processing flows that successfully send to Central.
	IndexReportProcessingOutcomeSuccess = "success"
	// IndexReportProcessingOutcomeNilReport marks flows that exit because the report was nil.
	IndexReportProcessingOutcomeNilReport = "nil_report"
	// IndexReportProcessingOutcomeBuildError marks flows that exit because the message could not be constructed.
	IndexReportProcessingOutcomeBuildError = "build_error"
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

// IndexReportEnqueueDurationMilliseconds measures how long Send spent waiting to enqueue reports.
var IndexReportEnqueueDurationMilliseconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.SensorSubsystem.String(),
		Name:      "virtual_machine_index_report_enqueue_duration_milliseconds",
		Help:      "Time spent (in ms) by Sensor while waiting to enqueue virtual machine index reports onto the indexReports channel",
		Buckets:   prometheus.ExponentialBuckets(0.5, 2, 13), // 0.5ms to ~4s
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

func init() {
	prometheus.MustRegister(
		IndexReportsReceived,
		IndexReportsSent,
		VirtualMachineIndexReportHandlingDurationMilliseconds,
		IndexReportProcessingDurationMilliseconds,
		IndexReportEnqueueDurationMilliseconds,
		IndexReportEnqueueBlockedTotal,
	)
}
