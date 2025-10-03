package env

var (
	// VirtualMachineVsockPort defines the port where the virtual machine relay will listen for incoming vsock
	// connections carrying virtual machine index reports
	VirtualMachineVsockPort = RegisterIntegerSetting("ROX_VIRTUAL_MACHINES_VSOCK_PORT", 1024).
		WithMinimum(1024).WithMaximum(65535)
)
