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

	// cluster health search fields
	ClusterStatus          = newFieldLabel("Cluster Status")
	SensorStatus           = newFieldLabel("Sensor Status")
	CollectorStatus        = newFieldLabel("Collector Status")
	AdmissionControlStatus = newFieldLabel("Admission Control Status")
	ScannerStatus          = newFieldLabel("Scanner Status")
	LastContactTime        = newFieldLabel("Last Contact")

	PolicyID           = newFieldLabel("Policy ID")
	Enforcement        = newFieldLabel("Enforcement")
	PolicyName         = newFieldLabel("Policy")
	PolicyCategoryName = newFieldLabel("Policy Category")
	PolicyCategoryID   = newFieldLabel("Policy Category ID")

	LifecycleStage = newFieldLabel("Lifecycle Stage")
	Description    = newFieldLabel("Description")
	Category       = newFieldLabel("Category")
	Severity       = newFieldLabel("Severity")
	Disabled       = newFieldLabel("Disabled")

	CVEID              = newFieldLabel("CVE ID")
	CVE                = newFieldLabel("CVE")
	CVEType            = newFieldLabel("CVE Type")
	CVEPublishedOn     = newFieldLabel("CVE Published On")
	CVECreatedTime     = newFieldLabel("CVE Created Time")
	CVESuppressed      = newFieldLabel("CVE Snoozed")
	CVESuppressExpiry  = newFieldLabel("CVE Snooze Expiry")
	CVSS               = newFieldLabel("CVSS")
	ImpactScore        = newFieldLabel("Impact Score")
	VulnerabilityState = newFieldLabel("Vulnerability State")

	Component                      = newFieldLabel("Component")
	ComponentID                    = newFieldLabel("Component ID")
	ComponentVersion               = newFieldLabel("Component Version")
	ComponentSource                = newFieldLabel("Component Source")
	ComponentLocation              = newFieldLabel("Component Location")
	ComponentTopCVSS               = newFieldLabel("Component Top CVSS")
	DockerfileInstructionKeyword   = newFieldLabel("Dockerfile Instruction Keyword")
	DockerfileInstructionValue     = newFieldLabel("Dockerfile Instruction Value")
	FirstImageOccurrenceTimestamp  = newFieldLabel("First Image Occurrence Timestamp")
	FirstSystemOccurrenceTimestamp = newFieldLabel("First System Occurrence Timestamp")
	HostIPC                        = newFieldLabel("Host IPC")
	HostNetwork                    = newFieldLabel("Host Network")
	HostPID                        = newFieldLabel("Host PID")
	ImageCreatedTime               = newFieldLabel("Image Created Time")
	ImageName                      = newFieldLabel("Image")
	ImageSHA                       = newFieldLabel("Image Sha")
	ImageSignatureFetchedTime      = newFieldLabel("Image Signature Fetched Time")
	ImageSignatureVerifiedBy       = newFieldLabel("Image Signature Verified By")
	ImageRegistry                  = newFieldLabel("Image Registry")
	ImageRemote                    = newFieldLabel("Image Remote")
	ImageScanTime                  = newFieldLabel("Image Scan Time")
	NodeScanTime                   = newFieldLabel("Node Scan Time")
	ImageOS                        = newFieldLabel("Image OS")
	ImageTag                       = newFieldLabel("Image Tag")
	ImageUser                      = newFieldLabel("Image User")
	ImageCommand                   = newFieldLabel("Image Command")
	ImageEntrypoint                = newFieldLabel("Image Entrypoint")
	ImageLabel                     = newFieldLabel("Image Label")
	ImageVolumes                   = newFieldLabel("Image Volumes")
	Fixable                        = newFieldLabel("Fixable")
	FixedBy                        = newFieldLabel("Fixed By")
	ClusterCVEFixedBy              = newFieldLabel("Cluster CVE Fixed By")
	ClusterCVEFixable              = newFieldLabel("Cluster CVE Fixable")
	FixableCVECount                = newFieldLabel("Fixable CVE Count")
	LastUpdatedTime                = newFieldLabel("Last Updated")
	ImageTopCVSS                   = newFieldLabel("Image Top CVSS")
	NodeTopCVSS                    = newFieldLabel("Node Top CVSS")

	// Deployment related fields
	AddCapabilities              = newFieldLabel("Add Capabilities")
	AllowPrivilegeEscalation     = newFieldLabel("Allow Privilege Escalation")
	AppArmorProfile              = newFieldLabel("AppArmor Profile")
	AutomountServiceAccountToken = newFieldLabel("Automount Service Account Token")
	DeploymentAnnotation         = newFieldLabel("Deployment Annotation")
	CPUCoresLimit                = newFieldLabel("CPU Cores Limit")
	CPUCoresRequest              = newFieldLabel("CPU Cores Request")
	ContainerID                  = newFieldLabel("Container ID")
	ContainerImageDigest         = newFieldLabel("Container Image Digest")
	ContainerName                = newFieldLabel("Container Name")
	DeploymentID                 = newFieldLabel("Deployment ID")
	DeploymentName               = newFieldLabel("Deployment")
	DeploymentLabel              = newFieldLabel("Deployment Label")
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
	AdministrationUsageTimestamp  = newFieldLabel("Administration Usage Timestamp")
	AdministrationUsageNodes      = newFieldLabel("Administration Usage Nodes")
	AdministrationUsageCPUUnits   = newFieldLabel("Administration Usage CPU Units")
	ClusterPriority               = newFieldLabel("Cluster Risk Priority")
	NamespacePriority             = newFieldLabel("Namespace Risk Priority")
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
	ServiceAccountLabel           = newFieldLabel("Service Account Label")
	ServiceAccountAnnotation      = newFieldLabel("Service Account Annotation")
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

	AlertID        = newFieldLabel("Alert ID")
	Violation      = newFieldLabel("Violation")
	ViolationState = newFieldLabel("Violation State")
	ViolationTime  = newFieldLabel("Violation Time")
	Tag            = newFieldLabel("Tag")

	// Pod Search fields
	PodUID   = newFieldLabel("Pod UID")
	PodID    = newFieldLabel("Pod ID")
	PodName  = newFieldLabel("Pod Name")
	PodLabel = newFieldLabel("Pod Label")

	// ProcessIndicator Search fields
	ProcessID           = newFieldLabel("Process ID")
	ProcessExecPath     = newFieldLabel("Process Path")
	ProcessName         = newFieldLabel("Process Name")
	ProcessArguments    = newFieldLabel("Process Arguments")
	ProcessAncestor     = newFieldLabel("Process Ancestor")
	ProcessUID          = newFieldLabel("Process UID")
	ProcessCreationTime = newFieldLabel("Process Creation Time")

	// ProcessListeningOnPort Search fields
	Closed     = newFieldLabel("Closed")
	ClosedTime = newFieldLabel("Closed Time")

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

	ComplianceOperatorVersion     = newFieldLabel("Compliance Operator Version")
	ComplianceOperatorScanName    = newFieldLabel("Compliance Scan Name")
	ComplianceOperatorSeverity    = newFieldLabel("Compliance Rule Severity")
	ComplianceOperatorCheckStatus = newFieldLabel("Compliance Check Status")
	ComplianceOperatorRuleName    = newFieldLabel("Compliance Rule Name")
	ComplianceOperatorProfileName = newFieldLabel("Compliance Profile Name")
	ComplianceOperatorStandard    = newFieldLabel("Compliance Standard")
	ComplianceOperatorScanConfig  = newFieldLabel("Compliance Scan Config ID")

	// Node search fields
	Node             = newFieldLabel("Node")
	NodeID           = newFieldLabel("Node ID")
	OperatingSystem  = newFieldLabel("Operating System")
	ContainerRuntime = newFieldLabel("Container Runtime")
	NodeJoinTime     = newFieldLabel("Node Join Time")
	NodeLabel        = newFieldLabel("Node Label")
	NodeAnnotation   = newFieldLabel("Node Annotation")

	// Namespace Search Fields
	NamespaceID         = newFieldLabel("Namespace ID")
	Namespace           = newFieldLabel("Namespace")
	NamespaceAnnotation = newFieldLabel("Namespace Annotation")
	NamespaceLabel      = newFieldLabel("Namespace Label")

	// Role Search Fields
	RoleID         = newFieldLabel("Role ID")
	RoleName       = newFieldLabel("Role")
	RoleLabel      = newFieldLabel("Role Label")
	RoleAnnotation = newFieldLabel("Role Annotation")
	ClusterRole    = newFieldLabel("Cluster Role")

	// Role Binding Search Fields
	RoleBindingID         = newFieldLabel("Role Binding ID")
	RoleBindingName       = newFieldLabel("Role Binding")
	RoleBindingLabel      = newFieldLabel("Role Binding Label")
	RoleBindingAnnotation = newFieldLabel("Role Binding Annotation")

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
	// Count-based derived fields. These fields are supported only in pagination.
	// The derived fields depending of fields with map and scalar data type array data structures are unsupported.
	NamespaceCount  = newDerivedFieldLabel("Namespace Count", NamespaceID, CountDerivationType)
	DeploymentCount = newDerivedFieldLabel("Deployment Count", DeploymentID, CountDerivationType)
	ImageCount      = newDerivedFieldLabel("Image Count", ImageSHA, CountDerivationType)
	NodeCount       = newDerivedFieldLabel("Node Count", NodeID, CountDerivationType)
	ComponentCount  = newDerivedFieldLabel("Component Count", ComponentID, CountDerivationType)
	CVECount        = newDerivedFieldLabel("CVE Count", CVEID, CountDerivationType)
	// Translative derived fields with reversed sorting. These fields are supported only in pagination.
	NodePriority       = newDerivedFieldLabel("Node Risk Priority", NodeRiskScore, SimpleReverseSortDerivationType)
	DeploymentPriority = newDerivedFieldLabel("Deployment Risk Priority", DeploymentRiskScore, SimpleReverseSortDerivationType)
	ImagePriority      = newDerivedFieldLabel("Image Risk Priority", ImageRiskScore, SimpleReverseSortDerivationType)
	ComponentPriority  = newDerivedFieldLabel("Component Risk Priority", ComponentRiskScore, SimpleReverseSortDerivationType)

	// External network sources fields
	DefaultExternalSource = newFieldLabel("Default External Source")

	// Report configurations search fields
	ReportName     = newFieldLabel("Report Name")
	ReportType     = newFieldLabel("Report Type")
	ReportConfigID = newFieldLabel("Report Configuration ID")

	// Resource alerts search fields
	ResourceName = newFieldLabel("Resource")
	ResourceType = newFieldLabel("Resource Type")

	// Vulnerability Watch Request fields
	RequestName                 = newFieldLabel("Request Name")
	RequestStatus               = newFieldLabel("Request Status")
	ExpiredRequest              = newFieldLabel("Expired Request")
	ExpiryType                  = newFieldLabel("Expiry Type")
	RequestExpiryTime           = newFieldLabel("Request Expiry Time")
	RequestExpiresWhenFixed     = newFieldLabel("Request Expires When Fixed")
	RequestedVulnerabilityState = newFieldLabel("Requested Vulnerability State")
	UserID                      = newFieldLabel("User ID")
	UserName                    = newFieldLabel("User Name")
	ImageRegistryScope          = newFieldLabel("Image Registry Scope")
	ImageRemoteScope            = newFieldLabel("Image Remote Scope")
	ImageTagScope               = newFieldLabel("Image Tag Scope")

	ComplianceDomainID             = newFieldLabel("Compliance Domain ID")
	ComplianceRunID                = newFieldLabel("Compliance Run ID")
	ComplianceRunFinishedTimestamp = newFieldLabel("Compliance Run Finished Timestamp")

	// Resource Collection fields
	CollectionID         = newFieldLabel("Collection ID")
	CollectionName       = newFieldLabel("Collection Name")
	EmbeddedCollectionID = newFieldLabel("Embedded Collection ID")

	// Group fields
	GroupAuthProvider = newFieldLabel("Group Auth Provider")
	GroupKey          = newFieldLabel("Group Key")
	GroupValue        = newFieldLabel("Group Value")

	// API Token fields
	Expiration = newFieldLabel("Expiration")
	Revoked    = newFieldLabel("Revoked")

	// Version fields
	Version               = newFieldLabel("Version")
	MinSequenceNumber     = newFieldLabel("Minimum Sequence Number")
	CurrentSequenceNumber = newFieldLabel("Current Sequence Number")
	LastPersistedTime     = newFieldLabel("Last Persisted")

	// Blob store fields
	BlobName             = newFieldLabel("Blob Name")
	BlobLength           = newFieldLabel("Blob Length")
	BlobModificationTime = newFieldLabel("Blob Modified On")

	// Report Metadata fields
	ReportState              = newFieldLabel("Report State")
	ReportQueuedTime         = newFieldLabel("Report Init Time")
	ReportCompletionTime     = newFieldLabel("Report Completion Time")
	ReportRequestType        = newFieldLabel("Report Request Type")
	ReportNotificationMethod = newFieldLabel("Report Notification Method")

	// Event fields.
	EventDomain     = newFieldLabel("Event Domain")
	EventType       = newFieldLabel("Event Type")
	EventLevel      = newFieldLabel("Event Level")
	EventOccurrence = newFieldLabel("Event Occurrence")

	// Test Search Fields
	TestKey               = newFieldLabel("Test Key")
	TestKey2              = newFieldLabel("Test Key 2")
	TestName              = newFieldLabel("Test Name")
	TestString            = newFieldLabel("Test String")
	TestStringSlice       = newFieldLabel("Test String Slice")
	TestBool              = newFieldLabel("Test Bool")
	TestUint64            = newFieldLabel("Test Uint64")
	TestInt64             = newFieldLabel("Test Int64")
	TestInt32Slice        = newFieldLabel("Test Int32 Slice")
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

	TestGrandparentID        = newFieldLabel("Test Grandparent ID")
	TestGrandparentVal       = newFieldLabel("Test Grandparent Val")
	TestGrandparentEmbedded  = newFieldLabel("Test Grandparent Embedded")
	TestGrandparentEmbedded2 = newFieldLabel("Test Grandparent Embedded2")
	TestGrandparentRiskScore = newFieldLabel("Test Grandparent Risk Score")
	TestParent1ID            = newFieldLabel("Test Parent1 ID")
	TestParent1Val           = newFieldLabel("Test Parent1 Val")
	TestParent1StringSlice   = newFieldLabel("Test Parent1 String Slice")
	TestChild1ID             = newFieldLabel("Test Child1 ID")
	TestChild1Val            = newFieldLabel("Test Child1 Val")
	TestGrandchild1ID        = newFieldLabel("Test Grandchild1 ID")
	TestGrandchild1Val       = newFieldLabel("Test Grandchild1 Val")
	TestGGrandchild1ID       = newFieldLabel("Test GGrandchild1 ID")
	TestGGrandchild1Val      = newFieldLabel("Test GGrandchild1 Val")
	TestG2Grandchild1ID      = newFieldLabel("Test G2Grandchild1 ID")
	TestG2Grandchild1Val     = newFieldLabel("Test G2Grandchild1 Val")
	TestG3Grandchild1ID      = newFieldLabel("Test G3Grandchild1 ID")
	TestG3Grandchild1Val     = newFieldLabel("Test G3Grandchild1 Val")
	TestParent2ID            = newFieldLabel("Test Parent2 ID")
	TestParent2Val           = newFieldLabel("Test Parent2 Val")
	TestChild2ID             = newFieldLabel("Test Child2 ID")
	TestChild2Val            = newFieldLabel("Test Child2 Val")
	TestParent3ID            = newFieldLabel("Test Parent3 ID")
	TestParent3Val           = newFieldLabel("Test Parent3 Val")
	TestParent4ID            = newFieldLabel("Test Parent4 ID")
	TestParent4Val           = newFieldLabel("Test Parent4 Val")
	TestChild1P4ID           = newFieldLabel("Test Child1P4 ID")
	TestChild1P4Val          = newFieldLabel("Test Child1P4 Val")

	TestShortCircuitID = newFieldLabel("Test ShortCircuit ID")

	// Derived test fields
	// The derived fields depending of fields with map and scalar data type array data structures are unsupported.
	TestGrandparentCount        = newDerivedFieldLabel("Test Grandparent Count", TestGrandparentID, CountDerivationType)
	TestParent1ValCount         = newDerivedFieldLabel("Test Parent1 Val Count", TestParent1Val, CountDerivationType)
	TestParent1Count            = newDerivedFieldLabel("Test Parent1 Count", TestParent1ID, CountDerivationType)
	TestChild1Count             = newDerivedFieldLabel("Test Child1 Count", TestChild1ID, CountDerivationType)
	TestGrandParentPriority     = newDerivedFieldLabel("Test Grandparent Priority", TestGrandparentRiskScore, SimpleReverseSortDerivationType)
	TestNestedStringCount       = newDerivedFieldLabel("Test Nested String Count", TestNestedString, CountDerivationType)
	TestNestedString2Count      = newDerivedFieldLabel("Test Nested String 2 Count", TestNestedString2, CountDerivationType)
	TestParent1StringSliceCount = newDerivedFieldLabel("Test Parent1 String Slice Count", TestParent1StringSlice, CountDerivationType)
)

func init() {
	derivedFields = set.NewStringSet()
	derivationsByField = make(map[string]map[string]DerivationType)
	for k, metadata := range allFieldLabels {
		if metadata != nil {
			derivedFields.Add(strings.ToLower(k))
			derivedFromLower := strings.ToLower(string(metadata.DerivedFrom))
			subMap, exists := derivationsByField[derivedFromLower]
			if !exists {
				subMap = make(map[string]DerivationType)
				derivationsByField[derivedFromLower] = subMap
			}
			subMap[k] = metadata.DerivationType
		}
	}
}

var (
	allFieldLabels     = make(map[string]*DerivedFieldLabelMetadata)
	derivationsByField map[string]map[string]DerivationType
	derivedFields      set.StringSet
)

// IsValidFieldLabel returns whether this is a known, valid field label.
func IsValidFieldLabel(s string) bool {
	_, ok := allFieldLabels[strings.ToLower(s)]
	return ok
}

// GetFieldsDerivedFrom gets the fields derived from the given search field.
func GetFieldsDerivedFrom(s string) map[string]DerivationType {
	return derivationsByField[strings.ToLower(s)]
}

// IsDerivedField returns if the search field is a derived field or not.
func IsDerivedField(s string) bool {
	return derivedFields.Contains(strings.ToLower(s))
}

func newFieldLabelWithMetadata(s string, metadata *DerivedFieldLabelMetadata) FieldLabel {
	lowerS := strings.ToLower(s)
	if _, exists := allFieldLabels[lowerS]; exists {
		log.Fatalf("Field label %q has already been added", s)
	}
	allFieldLabels[lowerS] = metadata
	return FieldLabel(s)
}

func newFieldLabel(s string) FieldLabel {
	return newFieldLabelWithMetadata(s, nil)
}

func newDerivedFieldLabel(s string, derivedFrom FieldLabel, derivationType DerivationType) FieldLabel {
	return newFieldLabelWithMetadata(s, &DerivedFieldLabelMetadata{
		DerivedFrom:    derivedFrom,
		DerivationType: derivationType,
	})
}

func (f FieldLabel) String() string {
	return string(f)
}

// DerivedFieldLabelMetadata includes metadata showing that a field is derived.
type DerivedFieldLabelMetadata struct {
	DerivedFrom    FieldLabel
	DerivationType DerivationType
}

// DerivationType represents a type of derivation.
//
//go:generate stringer -type=DerivationType
type DerivationType int

// This block enumerates all supported derivation types.
const (
	CountDerivationType DerivationType = iota
	SimpleReverseSortDerivationType
)
