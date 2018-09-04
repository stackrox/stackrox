package search

// A FieldLabel is the label we use to refer to a search field, as a human-readable shortcut.
// For example, if someone in the UI wants to filter the namespace to just "stackrox",
// they would type "Namespace:stackrox", and "Namespace" would be the field label.
type FieldLabel string

// This block enumerates all valid FieldLabels.
const (
	Cluster   FieldLabel = "Cluster"
	ClusterID FieldLabel = "Cluster Id"
	Namespace FieldLabel = "Namespace"
	Label     FieldLabel = "Label"

	PolicyID    FieldLabel = "Policy Id"
	Enforcement FieldLabel = "Enforcement"
	PolicyName  FieldLabel = "Policy"
	Description FieldLabel = "Description"
	Category    FieldLabel = "Category"
	Severity    FieldLabel = "Severity"

	CVE                          FieldLabel = "CVE"
	CVSS                         FieldLabel = "CVSS"
	Component                    FieldLabel = "Component"
	DockerfileInstructionKeyword FieldLabel = "Dockerfile Instruction Keyword"
	DockerfileInstructionValue   FieldLabel = "Dockerfile Instruction Value"
	ImageCreatedTime             FieldLabel = "Image Created Time"
	ImageName                    FieldLabel = "Image"
	ImageSHA                     FieldLabel = "Image Sha"
	ImageRegistry                FieldLabel = "Image Registry"
	ImageRemote                  FieldLabel = "Image Remote"
	ImageScanTime                FieldLabel = "Image Scan Time"
	ImageTag                     FieldLabel = "Image Tag"

	CPUCoresLimit     FieldLabel = "CPU Cores Limit"
	CPUCoresRequest   FieldLabel = "CPU Cores Request"
	DeploymentID      FieldLabel = "Deployment Id"
	DeploymentName    FieldLabel = "Deployment"
	DeploymentType    FieldLabel = "Deployment Type"
	AddCapabilities   FieldLabel = "Add Capabilities"
	DropCapabilities  FieldLabel = "Drop Capabilities"
	EnvironmentKey    FieldLabel = "Environment Key"
	EnvironmentValue  FieldLabel = "Environment Value"
	ImagePullSecret   FieldLabel = "Image Pull Secret"
	MemoryLimit       FieldLabel = "Memory Limit (MB)"
	MemoryRequest     FieldLabel = "Memory Request (MB)"
	Privileged        FieldLabel = "Privileged"
	SecretID          FieldLabel = "Secret Id"
	SecretName        FieldLabel = "Secret"
	SecretPath        FieldLabel = "Secret Path"
	ServiceAccount    FieldLabel = "Service Account"
	VolumeName        FieldLabel = "Volume Name"
	VolumeSource      FieldLabel = "Volume Source"
	VolumeDestination FieldLabel = "Volume Destination"
	VolumeReadonly    FieldLabel = "Volume ReadOnly"
	VolumeType        FieldLabel = "Volume Type"

	Violation FieldLabel = "Violation"
	Stale     FieldLabel = "Stale"
)

func (f FieldLabel) String() string {
	return string(f)
}
