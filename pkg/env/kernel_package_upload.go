package env

var (
	// DisableKernelPackageUpload is set to true to signal that kernel support package uploads should be disabled.
	DisableKernelPackageUpload = RegisterBooleanSetting("ROX_DISABLE_KERNEL_PACKAGE_UPLOAD", false)
)
