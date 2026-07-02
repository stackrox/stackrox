package env

import "time"

var (
	// VirtualMachinesMaxConcurrentVsockConnections defines the maximum number of vsock connections handled in parallel.
	VirtualMachinesMaxConcurrentVsockConnections = RegisterIntegerSetting(
		"ROX_VIRTUAL_MACHINES_MAX_CONCURRENT_VSOCK_CONNECTIONS", 50).WithMinimum(1)

	// VirtualMachinesConcurrencyTimeout defines the wait time before dropping a connection, when it cannot be handled
	// due to the concurrency limit (ROX_VIRTUAL_MACHINES_MAX_CONCURRENT_VSOCK_CONNECTIONS) being reached.
	VirtualMachinesConcurrencyTimeout = registerDurationSetting(
		"ROX_VIRTUAL_MACHINES_VSOCK_CONCURRENCY_TIMEOUT", 5*time.Second)

	// VirtualMachinesVsockConnMaxSizeKB defines the maximum size of incoming vsock connections. The 16 MB default
	// allows connections carrying index reports with up to approximately 6400 packages.
	VirtualMachinesVsockConnMaxSizeKB = RegisterIntegerSetting("ROX_VIRTUAL_MACHINES_VSOCK_CONN_MAX_SIZE_KB", 16384)

	// VirtualMachinesVsockPort defines the port where the virtual machine relay will listen for incoming vsock
	// connections carrying virtual machine index reports.
	VirtualMachinesVsockPort = RegisterIntegerSetting("ROX_VIRTUAL_MACHINES_VSOCK_PORT", 818).
					WithMaximum(65535).WithMinimum(0)

	// VirtualMachinesIndexReportsBufferSize defines the buffer size for the channel receiving virtual machine
	// index reports before they are sent to Central.
	VirtualMachinesIndexReportsBufferSize = RegisterIntegerSetting("ROX_VIRTUAL_MACHINES_INDEX_REPORTS_BUFFER_SIZE", 100).
						WithMinimum(0)

	// VirtualMachinesRelayEnabledOnMasterNodes allows enabling the VM relay on master/control-plane nodes.
	// The default is false because these nodes typically do not host VMs and would otherwise retry forever.
	VirtualMachinesRelayEnabledOnMasterNodes = RegisterBooleanSetting("ROX_VIRTUAL_MACHINES_RELAY_ENABLED_ON_MASTER_NODES", false)

	// VMIndexReportRateLimit enables concurrency limiting for VM index reports when set to any
	// value greater than 0. The actual concurrent processing capacity is controlled by
	// ROX_VM_INDEX_REPORT_BUCKET_CAPACITY. Set to "0" to disable limiting (unlimited).
	VMIndexReportRateLimit = RegisterFloatSetting("ROX_VM_INDEX_REPORT_RATE_LIMIT", 0.3)

	// VMIndexReportBucketCapacity defines the total number of VM index reports that can be
	// processed concurrently across all sensors. Each sensor gets an equal share (1/N) of this
	// capacity. Tokens are returned automatically when processing completes, so throughput
	// tracks actual processing speed rather than a fixed rate.
	// Default: 200 tokens
	VMIndexReportBucketCapacity = RegisterIntegerSetting("ROX_VM_INDEX_REPORT_BUCKET_CAPACITY", 200).WithMinimum(1)

	// VMRelayMaxReportsPerMinute defines the maximum number of reports per minute per VSOCK ID
	// that the relay will forward to Sensor. Uses leaky bucket (no bursts).
	// Supports fractional values for finer control (e.g., 0.5 = 1 report every 2 minutes).
	// Set to "0" to disable rate limiting.
	// Default: 1.0 report per minute per VM
	VMRelayMaxReportsPerMinute = RegisterFloatSetting("ROX_VM_RELAY_MAX_REPORTS_PER_MINUTE", 1.0).WithMinimum(0.0)

	// VMRelayStaleAckThreshold defines how long since the last ACK before considering it stale.
	// If rate limited and ACK is stale, the report is dropped (not sent to Sensor).
	// If rate limited and ACK is recent, the report is silently dropped to protect Sensor/Central.
	// Default: 4 hours
	VMRelayStaleAckThreshold = registerDurationSetting("ROX_VM_RELAY_STALE_ACK_THRESHOLD", 4*time.Hour)

	// VMIndexReportRelayCacheSlots is the maximum number of VM index report payloads the compliance-side relay
	// retains in memory for UMH-driven retransmission (one slot per distinct VSOCK ID with a cached report).
	// Set to 0 to disable the cache.
	VMIndexReportRelayCacheSlots = RegisterIntegerSetting("ROX_VM_INDEX_REPORT_RELAY_CACHE_SLOTS", 100).WithMinimum(0)

	// VMIndexReportRelayCacheTTL is how long a cached VM index report payload may be kept before it is evicted
	// from the relay's retransmission cache.
	VMIndexReportRelayCacheTTL = registerDurationSetting("ROX_VM_INDEX_REPORT_RELAY_CACHE_TTL", 4*time.Hour)
)
