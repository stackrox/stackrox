package env

import (
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
)

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

	// VirtualMachinesTestMode enables test mode for VM load testing across all components.
	// When enabled:
	// - Relay: bypasses vsock CID validation for loopback testing
	// - Sensor: auto-generates VMs on-the-fly, skips VM store cleanup
	// - Central: auto-creates missing VMs when receiving index reports
	// Use only for load testing scenarios.
	VirtualMachinesTestMode = RegisterBooleanSetting("ROX_VM_TEST_MODE", false)
)

// IsVMTestModeEnabled returns true if VM test mode is enabled via environment variable
// AND this is not a release build. This ensures test mode cannot be accidentally enabled
// in production releases.
func IsVMTestModeEnabled() bool {
	return VirtualMachinesTestMode.BooleanSetting() && !buildinfo.ReleaseBuild
}
