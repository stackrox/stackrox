package env

import (
	"math"
	"time"
)

// maxPullResponseSizeKB is the largest KB value that doesn't overflow
// uint32 once converted to bytes. VirtualMachinesPullMaxResponseSizeKB
// feeds vsockclient.NewClient, which narrows the byte count to uint32 for
// the wire protocol's frame-length field.
const maxPullResponseSizeKB = math.MaxUint32 / 1024

var (
	// VirtualMachinesVsockPort defines the VSOCK port the pull-mode scraper dials
	// on each VM to reach roxagent's GetReport endpoint.
	VirtualMachinesVsockPort = RegisterIntegerSetting("ROX_VIRTUAL_MACHINES_VSOCK_PORT", 818).
					WithMaximum(65535).WithMinimum(0)

	// VirtualMachinesIndexReportsBufferSize defines the buffer size for the channel receiving virtual machine
	// index reports before they are sent to Central.
	VirtualMachinesIndexReportsBufferSize = RegisterIntegerSetting("ROX_VIRTUAL_MACHINES_INDEX_REPORTS_BUFFER_SIZE", 100).
						WithMinimum(0)

	// VMIndexReportRateLimit defines the maximum number of VM index reports per second that Central will accept
	// across all sensors that actually send VM index reports. Each such sensor gets an equal share (1/N) of this
	// global capacity. The split happens when a new client ID registers in the VM index report pipeline; there is
	// no additional selection or weighting mechanism at the moment.
	// Supports fractional rates (e.g., "0.5" for one request every 2 seconds).
	// Set to "0" to disable rate limiting (unlimited).
	//
	// As of ACS 4.9 & 4.10, the default size cluster should not exceed 1.0 requests per second.
	// As VM scanning is only one of potentially many other workloads, we set the default to 0.3 requests per second.
	// For larger clusters, the rate limit could be increased to up to 3.0 requests per second only if the
	// scanner-v4-matcher and the scanner-v4-db are able to handle the load!
	VMIndexReportRateLimit = RegisterFloatSetting("ROX_VM_INDEX_REPORT_RATE_LIMIT", 0.3)

	// VMIndexReportBucketCapacity defines the token bucket capacity for VM index report rate limiting.
	// This is the maximum number of requests that can be accepted in a burst before rate limiting kicks in.
	// For example, with capacity=15 and rate=3 req/sec, a sensor can send up to 15 requests instantly,
	// then must wait for 5 seconds for tokens to refill at the rate limit.
	// Default: 200 tokens
	VMIndexReportBucketCapacity = RegisterIntegerSetting("ROX_VM_INDEX_REPORT_BUCKET_CAPACITY", 200).WithMinimum(1)

	// VirtualMachinesScraperConcurrency defines the maximum number of VMs scraped
	// concurrently in a single poll cycle. Higher values reduce cycle wall-clock
	// time but increase network fan-out.
	VirtualMachinesScraperConcurrency = RegisterIntegerSetting("ROX_VIRTUAL_MACHINES_SCRAPER_CONCURRENCY", 20).
						WithMinimum(1)

	// VirtualMachinesPullMaxResponseSizeKB defines the maximum response size (in KB) that the
	// pull-mode scraper accepts from a VM agent. Default 16384 KB (16 MiB) allows index reports
	// with up to approximately 6400 packages. Capped at maxPullResponseSizeKB so an
	// operator-configured value can never overflow the uint32 byte count vsockclient.NewClient
	// expects; a value above the cap falls back to the default instead.
	VirtualMachinesPullMaxResponseSizeKB = RegisterIntegerSetting("ROX_VIRTUAL_MACHINES_PULL_MAX_RESPONSE_SIZE_KB", 16384).
						WithMinimum(1).WithMaximum(maxPullResponseSizeKB)

	// VirtualMachinesScraperPollInterval defines how often the pull-mode scraper
	// polls VMs for new reports.
	VirtualMachinesScraperPollInterval = registerDurationSetting("ROX_VIRTUAL_MACHINES_SCRAPER_POLL_INTERVAL", 4*time.Hour)

	// VirtualMachinesScraperTickInterval is how often VMScraper considers due VMs.
	// Independent of retry backoff so operators can slow the ticker without
	// retuning NACK and failure delays.
	VirtualMachinesScraperTickInterval = registerDurationSetting("ROX_VIRTUAL_MACHINES_SCRAPER_TICK_INTERVAL", 10*time.Second)

	// VirtualMachinesScraperInitialBackoff is the first retry delay after a
	// retryable scrape failure or Central NACK. Later retries double, capped at
	// min(poll interval, 30m).
	VirtualMachinesScraperInitialBackoff = registerDurationSetting("ROX_VIRTUAL_MACHINES_SCRAPER_INITIAL_BACKOFF", 10*time.Second)

	// VirtualMachinesScraperPerVMTimeout is the deadline for one Sensor-to-agent
	// call (GetReport or a mapping sync). Doing both can take twice as long.
	VirtualMachinesScraperPerVMTimeout = registerDurationSetting("ROX_VIRTUAL_MACHINES_SCRAPER_PER_VM_TIMEOUT", 30*time.Second)

	// VirtualMachinesScraperMandatoryRefreshInterval defines the maximum age of a VM's last forwarded
	// report before the scraper requests a full report regardless of whether roxagent reports it as
	// unchanged. This bounds how long a report can go unevaluated against newly published Scanner V4
	// vulnerability definitions, mirroring ROX_REPROCESSING_INTERVAL and ROX_NODE_SCANNING_INTERVAL's
	// role for image and node scanning.
	VirtualMachinesScraperMandatoryRefreshInterval = registerDurationSetting("ROX_VIRTUAL_MACHINES_SCRAPER_MANDATORY_REFRESH_INTERVAL", 4*time.Hour)

	// VirtualMachinesScraperSteadySpreadFraction is the fraction of the poll interval used as the
	// one-sided post-poll random band W for cadence reschedule (nextAttemptAt =
	// now + pollInterval + U(0, W)). Default 2/3. Internal Sensor env only; not Operator/Helm-exposed.
	// Valid range is [0.01, 1] so W is not effectively zero; out-of-range values fall back to the default.
	VirtualMachinesScraperSteadySpreadFraction = RegisterFloatSetting("ROX_VIRTUAL_MACHINES_SCRAPER_STEADY_SPREAD_FRACTION", 2.0/3).
							WithMinimum(0.01).WithMaximum(1)

	// VirtualMachinesAgentStaleAfter is the GetVM Active window after the last
	// successful scrape. Default 12h is larger than one healthy poll cycle
	// (4h + up to 2/3 spread ≈ 6.7h), so one retryable VSOCK miss stays Active.
	VirtualMachinesAgentStaleAfter = registerDurationSetting("ROX_VIRTUAL_MACHINES_AGENT_STALE_AFTER", 12*time.Hour)
)
