package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/metrics"
)

// IndexReportsMismatchingVsockCID is a counter for the number of virtual machine index reports whose reported vsock CID does not
// match the connection's vsock CID.
var IndexReportsMismatchingVsockCID = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_reports_mismatching_vsock_cid_total",
		Help:      "Total number of virtual machine index reports received by this Relay with mismatching vsock CID",
	},
)

// IndexReportsReceived is a counter for the number of virtual machine index reports received.
var IndexReportsReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_reports_received_total",
		Help:      "Total number of virtual machine index reports received by this Relay",
	},
)

// IndexReportsSentToSensor is a counter for the number of virtual machine index reports sent to sensor.
var IndexReportsSentToSensor = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_reports_sent_total",
		Help:      "Total number of virtual machine index reports sent to sensor by this Relay",
	},
	[]string{"failed"},
)

// ConnectionsAccepted is a counter for the number of connections accepted by this relay. A mismatch between
// this and IndexReportsReceived indicates issues reading or parsing data.
var ConnectionsAccepted = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_connections_accepted_total",
		Help:      "Total number of connections accepted by this Relay",
	},
)

// SemaphoreAcquisitionFailures is a counter for the number of times the connection-handling semaphore that limits
// concurrency could not be acquired. A likely and significant reason for that is that the maximum parallel connections
// were reached.
var SemaphoreAcquisitionFailures = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_sem_acquisition_failures_total",
		Help:      "Number of failed attempts to acquire connection-handling semaphore",
	},
	[]string{"reason"},
)

// SemaphoreHoldingSize is the current number of VM relay connections actively being handled after acquiring the connection semaphore.
var SemaphoreHoldingSize = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_sem_holding_size",
		Help:      "Number of connections being handled",
	})

// SemaphoreQueueSize is the number of VM relay connections waiting to acquire the connection semaphore.
var SemaphoreQueueSize = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_sem_queue_size",
		Help:      "Number of connections waiting to be handled",
	})

// VMIndexReportSendAttempts counts send attempts to Sensor by result.
var VMIndexReportSendAttempts = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_report_send_attempts_total",
		Help:      "Send attempts of VM index reports to Sensor partitioned by result",
	},
	[]string{"result"}, // success|failure
)

// VMIndexReportSendDurationSeconds observes per-attempt latency to Sensor by result.
var VMIndexReportSendDurationSeconds = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_report_send_duration_seconds",
		Help:      "Duration of VM index report send attempts to Sensor",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10),
	},
	[]string{"result"}, // success|failure
)

// ReportsRateLimited counts reports dropped by relay-side rate limiting.
var ReportsRateLimited = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_reports_rate_limited_total",
		Help:      "Reports dropped due to relay-side rate limiting",
	},
	[]string{"reason"}, // "normal", "stale_ack"
)

// AcksReceived counts ACK confirmations for VM index reports on the Relay VM index path.
// It is incremented when ACK callback handling runs (not from the Sensor receive loop directly).
// NACKs are tracked separately in the main compliance component where they're handled.
var AcksReceived = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_acks_received_total",
		Help:      "ACK confirmations for VM index reports recorded by Relay ACK callback handling on the VM index path",
	},
)

// VMIndexACKsFromSensor counts ACK/NACK responses received from Sensor for VM index reports.
// This metric is recorded when compliance.go handles ComplianceACK messages.
var VMIndexACKsFromSensor = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_index_acks_from_sensor_total",
		Help:      "ACK/NACK responses received from Sensor for VM index reports",
	},
	[]string{"action"}, // "ACK", "NACK"
)

// IndexReportCacheSlotsUsed is the current number of entries held in the VM index report payload cache.
var IndexReportCacheSlotsUsed = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_report_cache_slots_used",
		Help:      "Number of VM index report payloads currently stored in the Relay payload cache",
	},
)

// IndexReportCacheSlotsCapacity is the maximum number of entries the VM index report payload cache can hold.
var IndexReportCacheSlotsCapacity = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_report_cache_slots_capacity",
		Help:      "Maximum number of VM index report payloads the Relay payload cache can store",
	},
)

// IndexReportCacheResidencySeconds observes elapsed time from the most recent payload update
// (updatedAt) to cache removal for VM index report payloads.
var IndexReportCacheResidencySeconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_report_cache_residency_seconds",
		Help:      "For each VM index report payload removed from cache, elapsed time between its most recent update and removal",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10),
	},
)

// IndexReportCacheLifetimeSeconds observes elapsed time from the first payload insert
// for the current cache entry (firstUpdatedAt) to cache removal.
var IndexReportCacheLifetimeSeconds = prometheus.NewHistogram(
	prometheus.HistogramOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_report_cache_lifetime_seconds",
		Help:      "For each VM index report payload removed from cache, elapsed time between first insert and removal",
		Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10),
	},
)

// IndexReportCacheLookupsTotal counts payload cache lookups by hit or miss.
var IndexReportCacheLookupsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: metrics.PrometheusNamespace,
		Subsystem: metrics.ComplianceSubsystem.String(),
		Name:      "virtual_machine_relay_index_report_cache_lookups_total",
		Help:      "Lookups against the VM index report payload cache partitioned by result",
	},
	[]string{"result"}, // hit|miss
)

func init() {
	prometheus.MustRegister(
		IndexReportsMismatchingVsockCID,
		IndexReportsReceived,
		IndexReportsSentToSensor,
		ConnectionsAccepted,
		SemaphoreAcquisitionFailures,
		SemaphoreHoldingSize,
		SemaphoreQueueSize,
		VMIndexReportSendAttempts,
		VMIndexReportSendDurationSeconds,
		ReportsRateLimited,
		AcksReceived,
		VMIndexACKsFromSensor,
		IndexReportCacheSlotsUsed,
		IndexReportCacheSlotsCapacity,
		IndexReportCacheResidencySeconds,
		IndexReportCacheLifetimeSeconds,
		IndexReportCacheLookupsTotal,
	)
}
