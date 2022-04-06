package search

import (
	"strings"

	"github.com/stackrox/rox/pkg/set"
)

// A FieldLabel is the label we use to refer to a search field, as a human-readable shortcut.
// For example, if someone in the UI wants to filter the namespace to just "stackrox",
// they would type "Namespace:stackrox", and "Namespace" would be the field label.
type FieldLabel string

// This block enumerates all valid FieldLabels.
var (
	FieldLabelSet = set.NewStringSet()

	// DocID is a special value for document identifier in the Bleve index.
	// Every document we put in the index has identifier. In most cases we simply get this identifier from
	// entity.GetId(), unless the getter is overridden in the call to blevebindings-wrapper with --id-func argument.
	// It is possible to use DocID in sorting, see https://blevesearch.com/docs/Sorting/, and subsequently in
	// pagination with SearchAfter.
	DocID = newFieldLabel("_id")

	Cluster      = newFieldLabel("Cluster")
	ClusterID    = newFieldLabel("Cluster ID")
	ClusterLabel = newFieldLabel("Cluster Label")
	ClusterScope = newFieldLabel("Cluster Scope")
	Label        = newFieldLabel("Label")
	PodLabel     = newFieldLabel("Pod Label")

	// cluster health search fields
	ClusterStatus          = newFieldLabel("Cluster Status")
	SensorStatus           = newFieldLabel("Sensor Status")
	CollectorStatus        = newFieldLabel("Collector Status")
	AdmissionControlStatus = newFieldLabel("Admission Control Status")

	PolicyID       = newFieldLabel("Policy ID")
	Enforcement    = newFieldLabel("Enforcement")
	PolicyName     = newFieldLabel("Policy")
	LifecycleStage = newFieldLabel("Lifecycle Stage")
	Description    = newFieldLabel("Description")
	Category       = newFieldLabel("Category")
	Severity       = newFieldLabel("Severity")
	Disabled       = newFieldLabel("Disabled")

	CVE                = newFieldLabel("CVE")
	CVECount           = newFieldLabel("CVE Count")
	CVEType            = newFieldLabel("CVE Type")
	CVEPublishedOn     = newFieldLabel("CVE Published On")
	CVECreatedTime     = newFieldLabel("CVE Created Time")
	CVESuppressed      = newFieldLabel("CVE Snoozed")
	CVESuppressExpiry  = newFieldLabel("CVE Snooze Expiry")
	CVSS               = newFieldLabel("CVSS")
	ImpactScore        = newFieldLabel("Impact Score")
	VulnerabilityState = newFieldLabel("Vulnerability State")

	Component                     = newFieldLabel("Component")
	ComponentID                   = newFieldLabel("Component ID")
	ComponentCount                = newFieldLabel("Component Count")
	ComponentVersion              = newFieldLabel("Component Version")
	ComponentSource               = newFieldLabel("Component Source")
	ComponentLocation             = newFieldLabel("Component Location")
	ComponentTopCVSS              = newFieldLabel("Component Top CVSS")
	DockerfileInstructionKeyword  = newFieldLabel("Dockerfile Instruction Keyword")
	DockerfileInstructionValue    = newFieldLabel("Dockerfile Instruction Value")
	FirstImageOccurrenceTimestamp = newFieldLabel("First Image Occurrence Timestamp")
	HostIPC                       = newFieldLabel("Host IPC")
	HostNetwork                   = newFieldLabel("Host Network")
	HostPID                       = newFieldLabel("Host PID")
	ImageCreatedTime              = newFieldLabel("Image Created Time")
	ImageName                     = newFieldLabel("Image")
	ImageSHA                      = newFieldLabel("Image Sha")
	ImageSignatureFetchedTime     = newFieldLabel("Image Signature Fetched Time")
	ImageSignatureVerifiedBy      = newFieldLabel("Image Signature Verified By")
	ImageRegistry                 = newFieldLabel("Image Registry")
	ImageRemote                   = newFieldLabel("Image Remote")
	ImageScanTime                 = newFieldLabel("Image Scan Time")
	NodeScanTime                  = newFieldLabel("Node Scan Time")
	ImageOS                       = newFieldLabel("Image OS")
	ImageTag                      = newFieldLabel("Image Tag")
	ImageUser                     = newFieldLabel("Image User")
	ImageCommand                  = newFieldLabel("Image Command")
	ImageEntrypoint               = newFieldLabel("Image Entrypoint")
	ImageLabel                    = newFieldLabel("Image Label")
	ImageVolumes                  = newFieldLabel("Image Volumes")
	Fixable                       = newFieldLabel("Fixable")
	FixedBy                       = newFieldLabel("Fixed By")
	ClusterCVEFixedBy             = newFieldLabel("Cluster CVE Fixed By")
	ClusterCVEFixable             = newFieldLabel("Cluster CVE Fixable")
	FixableCVECount               = newFieldLabel("Fixable CVE Count")
	LastUpdatedTime               = newFieldLabel("Last Updated")
	ImageTopCVSS                  = newFieldLabel("Image Top CVSS")
	NodeTopCVSS                   = newFieldLabel("Node Top CVSS")

	// Deployment related fields
	AddCapabilities              = newFieldLabel("Add Capabilities")
	AppArmorProfile              = newFieldLabel("AppArmor Profile")
	AutomountServiceAccountToken = newFieldLabel("Automount Service Account Token")
	Annotation                   = newFieldLabel("Annotation")
	CPUCoresLimit                = newFieldLabel("CPU Cores Limit")
	CPUCoresRequest              = newFieldLabel("CPU Cores Request")
	ContainerID                  = newFieldLabel("Container ID")
	ContainerName                = newFieldLabel("Container Name")
	ContainerImageDigest         = newFieldLabel("Container Image Digest")
	DeploymentID                 = newFieldLabel("Deployment ID")
	DeploymentName               = newFieldLabel("Deployment")
	DeploymentType               = newFieldLabel("Deployment Type")
	DropCapabilities             = newFieldLabel("Drop Capabilities")
	EnvironmentKey               = newFieldLabel("Environment Key")
	EnvironmentValue             = newFieldLabel("Environment Value")
	EnvironmentVarSrc            = newFieldLabel("Environment Variable Source")
	ExposedNodePort              = newFieldLabel("Exposed Node Port")
	ExposingService              = newFieldLabel("Exposing Service")
	ExposingServicePort          = newFieldLabel("Exposing Service Port")
	ExposureLevel                = newFieldLabel("Exposure Level")
	ExternalIP                   = newFieldLabel("External IP")
	ExternalHostname             = newFieldLabel("External Hostname")
	ImagePullSecret              = newFieldLabel("Image Pull Secret")
	LivenessProbeDefined         = newFieldLabel("Liveness Probe Defined")
	MaxExposureLevel             = newFieldLabel("Max Exposure Level")
	MemoryLimit                  = newFieldLabel("Memory Limit (MB)")
	MemoryRequest                = newFieldLabel("Memory Request (MB)")
	MountPropagation             = newFieldLabel("Mount Propagation")
	OrchestratorComponent        = newFieldLabel("Orchestrator Component")
	// PolicyViolated is a fake search field to filter deployments that have violation.
	// This is handled/supported only by deployments sub-resolver of policy resolver.
	// Note that 'Policy Violated=false' is not yet supported.
	PolicyViolated = newFieldLabel("Policy Violated")
	Port           = newFieldLabel("Port")
	PortProtocol   = newFieldLabel("Port Protocol")
	// Priority is used in risk datastore internally.
	Priority                      = newFieldLabel("Priority")
	ClusterPriority               = newFieldLabel("Cluster Risk Priority")
	NamespacePriority             = newFieldLabel("Namespace Risk Priority")
	NodePriority                  = newFieldLabel("Node Risk Priority")
	DeploymentPriority            = newFieldLabel("Deployment Risk Priority")
	ImagePriority                 = newFieldLabel("Image Risk Priority")
	ComponentPriority             = newFieldLabel("Component Risk Priority")
	Privileged                    = newFieldLabel("Privileged")
	ProcessTag                    = newFieldLabel("Process Tag")
	ReadOnlyRootFilesystem        = newFieldLabel("Read Only Root Filesystem")
	Replicas                      = newFieldLabel("Replicas")
	ReadinessProbeDefined         = newFieldLabel("Readiness Probe Defined")
	SecretID                      = newFieldLabel("Secret ID")
	SecretName                    = newFieldLabel("Secret")
	SecretPath                    = newFieldLabel("Secret Path")
	SeccompProfileType            = newFieldLabel("Seccomp Profile Type")
	ServiceAccountName            = newFieldLabel("Service Account")
	ServiceAccountPermissionLevel = newFieldLabel("Service Account Permission Level")
	Created                       = newFieldLabel("Created")
	VolumeName                    = newFieldLabel("Volume Name")
	VolumeSource                  = newFieldLabel("Volume Source")
	VolumeDestination             = newFieldLabel("Volume Destination")
	VolumeReadonly                = newFieldLabel("Volume ReadOnly")
	VolumeType                    = newFieldLabel("Volume Type")
	TaintKey                      = newFieldLabel("Taint Key")
	TaintValue                    = newFieldLabel("Taint Value")
	TolerationKey                 = newFieldLabel("Toleration Key")
	TolerationValue               = newFieldLabel("Toleration Value")
	TolerationEffect              = newFieldLabel("Taint Effect")

	Violation      = newFieldLabel("Violation")
	ViolationState = newFieldLabel("Violation State")
	ViolationTime  = newFieldLabel("Violation Time")
	Tag            = newFieldLabel("Tag")

	// Pod Search fields
	PodUID  = newFieldLabel("Pod UID")
	PodID   = newFieldLabel("Pod ID")
	PodName = newFieldLabel("Pod Name")

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
	Node             = newFieldLabel("Node")
	NodeID           = newFieldLabel("Node ID")
	OperatingSystem  = newFieldLabel("Operating System")
	ContainerRuntime = newFieldLabel("Container Runtime")
	NodeJoinTime     = newFieldLabel("Node Join Time")

	// Namespace Search Fields
	NamespaceID         = newFieldLabel("Namespace ID")
	Namespace           = newFieldLabel("Namespace")
	NamespaceAnnotation = newFieldLabel("Namespace Annotation")

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

	// Inactive Deployment
	Inactive = newFieldLabel("Inactive Deployment")

	// Risk Search Fields
	RiskScore           = newFieldLabel("Risk Score")
	NodeRiskScore       = newFieldLabel("Node Risk Score")
	DeploymentRiskScore = newFieldLabel("Deployment Risk Score")
	ImageRiskScore      = newFieldLabel("Image Risk Score")
	ComponentRiskScore  = newFieldLabel("Component Risk Score")
	RiskSubjectType     = newFieldLabel("Risk Subject Type")

	PolicyLastUpdated = newFieldLabel("Policy Last Updated")

	// Following are helper fields used for sorting
	// For example, "SORTPolicyName" field should be used to sort policies when the query sort field is "PolicyName"
	SORTPolicyName     = newFieldLabel("SORT_Policy")
	SORTLifecycleStage = newFieldLabel("SORT_Lifecycle Stage")
	SORTEnforcement    = newFieldLabel("SORT_Enforcement")

	// Following are derived fields
	NamespaceCount  = newFieldLabel("Namespace Count")
	DeploymentCount = newFieldLabel("Deployment Count")
	ImageCount      = newFieldLabel("Image Count")
	NodeCount       = newFieldLabel("Node Count")

	// External network sources fields
	DefaultExternalSource = newFieldLabel("Default External Source")

	// Report configurations search fields
	ReportName = newFieldLabel("Report Name")
	ReportType = newFieldLabel("Report Type")

	// Resource alerts search fields
	ResourceName = newFieldLabel("Resource")
	ResourceType = newFieldLabel("Resource Type")

	// Vulnerability Watch Request fields
	RequestStatus               = newFieldLabel("Request Status")
	ExpiredRequest              = newFieldLabel("Expired Request")
	RequestExpiryTime           = newFieldLabel("Request Expiry Time")
	RequestExpiresWhenFixed     = newFieldLabel("Request Expires When Fixed")
	RequestedVulnerabilityState = newFieldLabel("Requested Vulnerability State")
	UserName                    = newFieldLabel("User Name")

	// Test Search Fields
	TestKey               = newFieldLabel("Test Key")
	TestName              = newFieldLabel("Test Name")
	TestString            = newFieldLabel("Test String")
	TestStringSlice       = newFieldLabel("Test String Slice")
	TestBool              = newFieldLabel("Test Bool")
	TestUint64            = newFieldLabel("Test Uint64")
	TestInt64             = newFieldLabel("Test Int64")
	TestInt64Slice        = newFieldLabel("Test Int64 Slice")
	TestFloat             = newFieldLabel("Test Float")
	TestLabels            = newFieldLabel("Test Labels")
	TestTimestamp         = newFieldLabel("Test Timestamp")
	TestEnum              = newFieldLabel("Test Enum")
	TestEnumSlice         = newFieldLabel("Test Enum Slice")
	TestNestedString      = newFieldLabel("Test Nested String")
	TestNestedString2     = newFieldLabel("Test Nested String 2")
	TestNestedBool        = newFieldLabel("Test Nested Bool")
	TestNestedBool2       = newFieldLabel("Test Nested Bool 2")
	TestNestedInt64       = newFieldLabel("Test Nested Int64")
	TestNested2Int64      = newFieldLabel("Test Nested Int64 2")
	TestOneofNestedString = newFieldLabel("Test Oneof Nested String")
)

func newFieldLabel(s string) FieldLabel {
	if added := FieldLabelSet.Add(s); !added {
		log.Fatalf("Field label %q has already been added", s)
	}
	FieldLabelSet.Add(strings.ToLower(s))
	return FieldLabel(s)
}

func (f FieldLabel) String() string {
	return string(f)
}
