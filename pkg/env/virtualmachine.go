package env

var (
	// VirtualMachinesVsockConnMaxSizeKB defines the maximum size of incoming vsock connections. The 4 MB default
	// allows connections carrying index reports with up to approximately 10000 packages.
	VirtualMachinesVsockConnMaxSizeKB = RegisterIntegerSetting("ROX_VIRTUAL_MACHINES_VSOCK_CONN_MAX_SIZE_KB", 4096)

	// VirtualMachinesVsockPort defines the port where the virtual machine relay will listen for incoming vsock
	// connections carrying virtual machine index reports.
	VirtualMachinesVsockPort = RegisterIntegerSetting("ROX_VIRTUAL_MACHINES_VSOCK_PORT", 1024).
					WithMinimum(1024).WithMaximum(65535)
)
