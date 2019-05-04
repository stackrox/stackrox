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

	Cluster      = newFieldLabel("Cluster")
	ClusterID    = newFieldLabel("Cluster ID")
	ClusterScope = newFieldLabel("Cluster Scope")
	Label        = newFieldLabel("Label")
	PodLabel     = newFieldLabel("Pod Label")

	PolicyID       = newFieldLabel("Policy ID")
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
	ImageUser                    = newFieldLabel("Image User")
	ImageCommand                 = newFieldLabel("Image Command")
	ImageEntrypoint              = newFieldLabel("Image Entrypoint")
	ImageVolumes                 = newFieldLabel("Image Volumes")
	FixedBy                      = newFieldLabel("Fixed By")
	LastUpdatedTime              = newFieldLabel("Last Updated")

	AddCapabilities        = newFieldLabel("Add Capabilities")
	Annotation             = newFieldLabel("Annotation")
	CPUCoresLimit          = newFieldLabel("CPU Cores Limit")
	CPUCoresRequest        = newFieldLabel("CPU Cores Request")
	ContainerID            = newFieldLabel("Container ID")
	DeploymentID           = newFieldLabel("Deployment ID")
	DeploymentName         = newFieldLabel("Deployment")
	DeploymentType         = newFieldLabel("Deployment Type")
	DropCapabilities       = newFieldLabel("Drop Capabilities")
	EnvironmentKey         = newFieldLabel("Environment Key")
	EnvironmentValue       = newFieldLabel("Environment Value")
	ExposedNodePort        = newFieldLabel("Exposed Node Port")
	ExposingService        = newFieldLabel("Exposing Service")
	ExposingServicePort    = newFieldLabel("Exposing Service Port")
	ExposureLevel          = newFieldLabel("Exposure Level")
	ExternalIP             = newFieldLabel("External IP")
	ExternalHostname       = newFieldLabel("External Hostname")
	ImagePullSecret        = newFieldLabel("Image Pull Secret")
	MaxExposureLevel       = newFieldLabel("Max Exposure Level")
	MemoryLimit            = newFieldLabel("Memory Limit (MB)")
	MemoryRequest          = newFieldLabel("Memory Request (MB)")
	Port                   = newFieldLabel("Port")
	PortProtocol           = newFieldLabel("Port Protocol")
	Privileged             = newFieldLabel("Privileged")
	ReadOnlyRootFilesystem = newFieldLabel("Read Only Root Filesystem")
	SecretID               = newFieldLabel("Secret ID")
	SecretName             = newFieldLabel("Secret")
	SecretPath             = newFieldLabel("Secret Path")
	ServiceAccountName     = newFieldLabel("Service Account")
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
	ViolationTime  = newFieldLabel("Violation Time")

	// ProcessIndicator Search fields
	ProcessID        = newFieldLabel("Process ID")
	ProcessExecPath  = newFieldLabel("Process Path")
	ProcessName      = newFieldLabel("Process Name")
	ProcessArguments = newFieldLabel("Process Arguments")
	ProcessAncestor  = newFieldLabel("Process Ancestor")
	ProcessUID       = newFieldLabel("Process UID")

	// Secret search fields
	SecretType       = newFieldLabel("Secret Type")
	SecretExpiration = newFieldLabel("Cert Expiration")
	SecretRegistry   = newFieldLabel("Image Pull Secret Registry")

	// Compliance search fields
	Standard   = newFieldLabel("Standard")
	StandardID = newFieldLabel("Standard ID")

	ControlGroupID = newFieldLabel("Control Group ID")
	ControlGroup   = newFieldLabel("Control Group")

	ControlID = newFieldLabel("Control ID")
	Control   = newFieldLabel("Control")

	// Node search fields
	Node   = newFieldLabel("Node")
	NodeID = newFieldLabel("Node ID")

	// Namespace Search Fields
	NamespaceID = newFieldLabel("Namespace ID")
	Namespace   = newFieldLabel("Namespace")

	// Role Search Fields
	RoleID      = newFieldLabel("Role ID")
	RoleName    = newFieldLabel("Role")
	ClusterRole = newFieldLabel("Cluster Role")

	// Role Binding Search Fields
	RoleBindingID   = newFieldLabel("Role Binding ID")
	RoleBindingName = newFieldLabel("Role Binding")

	// Subject search fields
	SubjectKind = newFieldLabel("Subject Kind")
	SubjectName = newFieldLabel("Subject")

	// General
	CreatedTime = newFieldLabel("Created Time")
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
