package vmhelpers

// Prometheus metric names and label keys must stay aligned with product code:
//   - compliance/virtualmachines/relay/metrics/metrics.go
//   - sensor/common/virtualmachine/metrics/metrics.go

// prometheusNamespace is the metric name prefix shared by StackRox VM-related Prometheus series in tests.
const prometheusNamespace = "rox"

// Compliance relay series (subsystem "compliance" in product metrics).
const (
	MetricComplianceRelayConnectionsAcceptedTotal          = prometheusNamespace + "_compliance_virtual_machine_relay_connections_accepted_total"
	MetricComplianceRelayIndexReportsReceivedTotal         = prometheusNamespace + "_compliance_virtual_machine_relay_index_reports_received_total"
	MetricComplianceRelayIndexReportsSentTotal             = prometheusNamespace + "_compliance_virtual_machine_relay_index_reports_sent_total"
	MetricComplianceRelayIndexReportsMismatchingVsockTotal = prometheusNamespace + "_compliance_virtual_machine_relay_index_reports_mismatching_vsock_cid_total"
	MetricComplianceRelaySemaphoreAcquisitionFailuresTotal = prometheusNamespace + "_compliance_virtual_machine_relay_sem_acquisition_failures_total"

	// Expected but not yet implemented: compliance relay should track ACKs propagated back from Sensor.
	MetricComplianceRelayIndexReportAcksReceivedTotal = prometheusNamespace + "_compliance_virtual_machine_relay_index_report_acks_received_total"
)

// Sensor VM index series (subsystem "sensor").
const (
	MetricSensorVMIndexReportsReceivedTotal          = prometheusNamespace + "_sensor_virtual_machine_index_reports_received_total"
	MetricSensorVMIndexReportsSentTotal              = prometheusNamespace + "_sensor_virtual_machine_index_reports_sent_total"
	MetricSensorVMIndexReportAcksReceivedTotal       = prometheusNamespace + "_sensor_virtual_machine_index_report_acks_received_total"
	MetricSensorVMIndexReportEnqueueBlockedTotal     = prometheusNamespace + "_sensor_virtual_machine_index_report_enqueue_blocked_total"
	MetricSensorVMIndexReportBlockingEnqueueDuration = prometheusNamespace + "_sensor_virtual_machine_index_report_blocking_enqueue_duration_milliseconds"
	MetricSensorVMIndexReportHandlingDuration        = prometheusNamespace + "_sensor_virtual_machine_index_report_handling_duration_milliseconds"
	MetricSensorVMIndexReportProcessingDuration      = prometheusNamespace + "_sensor_virtual_machine_index_report_processing_duration_milliseconds"
)

// Prometheus label keys used in scrape text for the above vectors (must match registration).
const (
	LabelFailed = "failed"
	LabelStatus = "status"
	LabelAction = "action"
)

// Sensor IndexReportsSent status label values from sensor/common/virtualmachine/metrics/metrics.go
const (
	SensorIndexReportStatusCentralNotReady = "central not ready"
	SensorIndexReportStatusError           = "error"
	SensorIndexReportStatusSuccess         = "success"
)
