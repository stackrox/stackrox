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

	// VirtualMachinesRelayTestMode bypasses vsock CID validation in the relay. Use only for load testing scenarios.
	VirtualMachinesRelayTestMode = RegisterBooleanSetting("ROX_VM_RELAY_TEST_MODE", false)

	// VirtualMachinesSensorTestMode enables test mode for artificial load testing in sensor.
	// When enabled, sensor prepopulates the VM store with fake VMs during initialization.
	VirtualMachinesSensorTestMode = RegisterBooleanSetting("ROX_VM_SENSOR_TEST_MODE", false)

	// VirtualMachinesSensorTestVMCount configures the number of VMs to prepopulate in test mode.
	// VMs are assigned sequential CIDs starting from 3. Max 100000 VMs.
	VirtualMachinesSensorTestVMCount = RegisterIntegerSetting("ROX_VM_SENSOR_TEST_VM_COUNT", 100000).
						WithMinimum(1).WithMaximum(100000)
)
