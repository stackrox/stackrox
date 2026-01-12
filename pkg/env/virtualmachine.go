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
	// Supports fractional rates (e.g., "0.5" for one request every 2 seconds).
	// Set to "0" to disable rate limiting (unlimited).
	//
	// As of ACS 4.9 & 4.10, the default size cluster should not exceed 1.0 requests per second.
	// For larger clusters, the rate limit could be increased to up to 3.0 requests per second only if the
	// scanner-v4-matcher and the scanner-v4-db are able to handle the load!
	VMIndexReportRateLimit = RegisterSetting("ROX_VM_INDEX_REPORT_RATE_LIMIT", WithDefault("1.0"))

	// VMIndexReportBucketCapacity defines the token bucket capacity for VM index report rate limiting.
	// This is the maximum number of requests that can be accepted in a burst before rate limiting kicks in.
	// For example, with capacity=15 and rate=3 req/sec, a sensor can send up to 15 requests instantly,
	// then must wait for 5 seconds for tokens to refill at the rate limit.
	// Default: 5 tokens
	VMIndexReportBucketCapacity = RegisterIntegerSetting("ROX_VM_INDEX_REPORT_BUCKET_CAPACITY", 5).WithMinimum(1)
)
