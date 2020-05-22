package fieldnames

var (
	numFields int
)

// This block enumerates all known field names.
var (
	AddCaps                = newFieldName("Add Capabilities")
	CVE                    = newFieldName("CVE")
	CVSS                   = newFieldName("CVSS")
	ContainerCPULimit      = newFieldName("Container CPU Limit")
	ContainerCPURequest    = newFieldName("Container CPU Request")
	ContainerMemLimit      = newFieldName("Container Memory Limit")
	ContainerMemRequest    = newFieldName("Container Memory Request")
	DisallowedAnnotation   = newFieldName("Disallowed Annotation")
	DisallowedImageLabel   = newFieldName("Disallowed Image Label")
	DockerfileLine         = newFieldName("Dockerfile Line")
	DropCaps               = newFieldName("Drop Capabilities")
	EnvironmentVariable    = newFieldName("Environment Variable")
	FixedBy                = newFieldName("Fixed By")
	ImageAge               = newFieldName("Image Age")
	ImageComponent         = newFieldName("Image Component")
	ImageRegistry          = newFieldName("Image Registry")
	ImageRemote            = newFieldName("Image Remote")
	ImageScanAge           = newFieldName("Image Scan Age")
	ImageTag               = newFieldName("Image Tag")
	MinimumRBACPermissions = newFieldName("Minimum RBAC Permissions")
	Port                   = newFieldName("Port")
	PortExposure           = newFieldName("Port Exposure Method")
	Privileged             = newFieldName("Privileged")
	ProcessAncestor        = newFieldName("Process Ancestor")
	ProcessArguments       = newFieldName("Process Arguments")
	ProcessName            = newFieldName("Process Name")
	ProcessUID             = newFieldName("Process UID")
	Protocol               = newFieldName("Protocol")
	ReadOnlyRootFS         = newFieldName("Read-Only Root Filesystem")
	RequiredAnnotation     = newFieldName("Required Annotation")
	RequiredImageLabel     = newFieldName("Required Image Label")
	RequiredLabel          = newFieldName("Required Label")
	UnscannedImage         = newFieldName("Unscanned Image")
	VolumeDestination      = newFieldName("Volume Destination")
	VolumeName             = newFieldName("Volume Name")
	VolumeSource           = newFieldName("Volume Source")
	VolumeType             = newFieldName("Volume Type")
	WhitelistsEnabled      = newFieldName("Unexpected Process Executed")
	// TODO(rc) check volume type is hostpath and not read only
	WritableHostMount = newFieldName("Writable Host Mount")
	WritableVolume    = newFieldName("Writable Volume")
)

func newFieldName(field string) string {
	numFields++
	return field
}

// Count returns the number of known field names. It's useful for testing.
func Count() int {
	return numFields
}
