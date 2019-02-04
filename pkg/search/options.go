package search

import (
	"github.com/stackrox/rox/pkg/set"
)

// A FieldLabel is the label we use to refer to a search field, as a human-readable shortcut.
// For example, if someone in the UI wants to filter the namespace to just "stackrox",
// they would type "Namespace:stackrox", and "Namespace" would be the field label.
type FieldLabel string

// This block enumerates all valid FieldLabels.
var (
	FieldLabelSet = set.NewStringSet()

	Cluster   = newFieldLabel("Cluster")
	ClusterID = newFieldLabel("Cluster Id")
	Namespace = newFieldLabel("Namespace")
	Label     = newFieldLabel("Label")
	PodLabel  = newFieldLabel("PodLabel")

	PolicyID       = newFieldLabel("Policy Id")
	Enforcement    = newFieldLabel("Enforcement")
	PolicyName     = newFieldLabel("Policy")
	LifecycleStage = newFieldLabel("Lifecycle Stage")
	Description    = newFieldLabel("Description")
	Category       = newFieldLabel("Category")
	Severity       = newFieldLabel("Severity")

	CVE                          = newFieldLabel("CVE")
	CVELink                      = newFieldLabel("CVE Link")
	CVSS                         = newFieldLabel("CVSS")
	Component                    = newFieldLabel("Component")
	ComponentVersion             = newFieldLabel("Component Version")
	DockerfileInstructionKeyword = newFieldLabel("Dockerfile Instruction Keyword")
	DockerfileInstructionValue   = newFieldLabel("Dockerfile Instruction Value")
	ImageCreatedTime             = newFieldLabel("Image Created Time")
	ImageName                    = newFieldLabel("Image")
	ImageSHA                     = newFieldLabel("Image Sha")
	ImageRegistry                = newFieldLabel("Image Registry")
	ImageRemote                  = newFieldLabel("Image Remote")
	ImageScanTime                = newFieldLabel("Image Scan Time")
	ImageTag                     = newFieldLabel("Image Tag")

	Annotation             = newFieldLabel("Annotation")
	CPUCoresLimit          = newFieldLabel("CPU Cores Limit")
	CPUCoresRequest        = newFieldLabel("CPU Cores Request")
	ContainerID            = newFieldLabel("Container Id")
	DeploymentID           = newFieldLabel("Deployment Id")
	DeploymentName         = newFieldLabel("Deployment")
	DeploymentType         = newFieldLabel("Deployment Type")
	AddCapabilities        = newFieldLabel("Add Capabilities")
	DropCapabilities       = newFieldLabel("Drop Capabilities")
	ReadOnlyRootFilesystem = newFieldLabel("Read Only Root Filesystem")
	EnvironmentKey         = newFieldLabel("Environment Key")
	EnvironmentValue       = newFieldLabel("Environment Value")
	ImagePullSecret        = newFieldLabel("Image Pull Secret")
	MemoryLimit            = newFieldLabel("Memory Limit (MB)")
	MemoryRequest          = newFieldLabel("Memory Request (MB)")
	Port                   = newFieldLabel("Port")
	PortProtocol           = newFieldLabel("Port Protocol")
	Privileged             = newFieldLabel("Privileged")
	SecretID               = newFieldLabel("Secret Id")
	SecretName             = newFieldLabel("Secret")
	SecretPath             = newFieldLabel("Secret Path")
	ServiceAccount         = newFieldLabel("Service Account")
	VolumeName             = newFieldLabel("Volume Name")
	VolumeSource           = newFieldLabel("Volume Source")
	VolumeDestination      = newFieldLabel("Volume Destination")
	VolumeReadonly         = newFieldLabel("Volume ReadOnly")
	VolumeType             = newFieldLabel("Volume Type")
	TaintKey               = newFieldLabel("Taint Key")
	TaintValue             = newFieldLabel("Taint Value")
	TolerationKey          = newFieldLabel("Toleration Key")
	TolerationValue        = newFieldLabel("Toleration Value")
	TolerationEffect       = newFieldLabel("Taint Effect")

	Violation      = newFieldLabel("Violation")
	ViolationState = newFieldLabel("Violation State")

	// ProcessIndicator Search fields
	ProcessID        = newFieldLabel("Process ID")
	ProcessExecPath  = newFieldLabel("Process Path")
	ProcessName      = newFieldLabel("Process Name")
	ProcessArguments = newFieldLabel("Process Arguments")
	ProcessAncestor  = newFieldLabel("Process Ancestor")

	// Secret search fields
	SecretType       = newFieldLabel("Secret Type")
	SecretExpiration = newFieldLabel("Cert Expiration")

	// Compliance search fields
	Standard   = newFieldLabel("Standard")
	StandardID = newFieldLabel("Standard Id")

	ControlGroupID = newFieldLabel("Control Group Id")
	ControlGroup   = newFieldLabel("Control Group")

	ControlID = newFieldLabel("Control Id")
	Control   = newFieldLabel("Control")

	// Node search fields
	Node = newFieldLabel("Node")
)

func newFieldLabel(s string) FieldLabel {
	if added := FieldLabelSet.Add(s); !added {
		log.Fatalf("Field label %q has already been added", s)
	}
	return FieldLabel(s)
}

func (f FieldLabel) String() string {
	return string(f)
}
