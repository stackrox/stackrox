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

	// VMIndexReportRateLimit defines the maximum number of VM index reports per second that Central will accept
	// across all sensors. Each sensor gets an equal share (1/N) of this global capacity.
	// Set to 0 to disable rate limiting (unlimited).
	// Default: 0 (disabled) for safe rollout; recommended production value: 3
	VMIndexReportRateLimit = RegisterIntegerSetting("ROX_VM_INDEX_REPORT_RATE_LIMIT", 0).WithMinimum(0)

	// VMIndexReportBurstDuration defines the burst window duration for VM index report rate limiting.
	// This allows temporary bursts above the sustained rate limit.
	// For example, with rate=3 req/sec and burst=5s, a sensor can send up to 15 reports in a 5-second window.
	// Default: 5 seconds
	VMIndexReportBurstDuration = registerDurationSetting("ROX_VM_INDEX_REPORT_BURST_DURATION", 5*time.Second)
)
