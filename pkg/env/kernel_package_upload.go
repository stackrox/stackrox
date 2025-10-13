package env

var (
	// EnableKernelPackageUpload is set to true to signal that kernel support package uploads should be supported.
	EnableKernelPackageUpload = RegisterBooleanSetting("ROX_ENABLE_KERNEL_PACKAGE_UPLOAD", true)
)
