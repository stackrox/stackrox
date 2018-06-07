package search

// The following strings are the internal representation to their option value
const (
	Cluster    = "Cluster"
	Namespace  = "Namespace"
	LabelKey   = "Label Key"
	LabelValue = "Label Value"

	PolicyID    = "Policy Id"
	Enforcement = "Enforcement"
	PolicyName  = "Policy Name"
	Description = "Description"
	Category    = "Category"
	Severity    = "Severity"

	CVE                          = "CVE"
	CVSS                         = "CVSS"
	Component                    = "Component"
	DockerfileInstructionKeyword = "Dockerfile Instruction Keyword"
	DockerfileInstructionValue   = "Dockerfile Instruction Value"
	ImageName                    = "Image Name"
	ImageSHA                     = "Image Sha"
	ImageRegistry                = "Image Registry"
	ImageRemote                  = "Image Remote"
	ImageTag                     = "Image Tag"

	DeploymentID      = "Deployment Id"
	DeploymentName    = "Deployment Name"
	DeploymentType    = "Deployment Type"
	AddCapabilities   = "Add Capabilities"
	DropCapabilities  = "Drop Capabilities"
	EnvironmentKey    = "Environment Key"
	EnvironmentValue  = "Environment Value"
	Privileged        = "Privileged"
	SecretName        = "Secret Name"
	SecretPath        = "Secret Path"
	VolumeName        = "Volume Name"
	VolumeSource      = "Volume Source"
	VolumeDestination = "Volume Destination"
	VolumeReadonly    = "Volume ReadOnly"
	VolumeType        = "Volume Type"

	Violation = "Violation"
	Stale     = "Stale"
)
