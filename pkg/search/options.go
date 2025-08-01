package search

import (
	"strings"

	"github.com/stackrox/rox/pkg/postgres"
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

	Cluster                  = newFieldLabel("Cluster")
	ClusterID                = newFieldLabel("Cluster ID")
	ClusterLabel             = newFieldLabel("Cluster Label")
	ClusterScope             = newFieldLabel("Cluster Scope")
	ClusterType              = newFieldLabel("Cluster Type")
	ClusterDiscoveredTime    = newFieldLabel("Cluster Discovered Time")
	ClusterPlatformType      = newFieldLabel("Cluster Platform Type")
	ClusterKubernetesVersion = newFieldLabel("Cluster Kubernetes Version")

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
	NVDCVSS            = newFieldLabel("NVD CVSS")
	ImpactScore        = newFieldLabel("Impact Score")
	VulnerabilityState = newFieldLabel("Vulnerability State")
	CVEOrphaned        = newFieldLabel("CVE Orphaned")
	CVEOrphanedTime    = newFieldLabel("CVE Orphaned Time")
	EPSSProbablity     = newFieldLabel("EPSS Probability")
	AdvisoryName       = newFieldLabel("Advisory Name")
	AdvisoryLink       = newFieldLabel("Advisory Link")

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
	ImageCVECount                  = newFieldLabel("Image CVE Count")
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
	ImageID                        = newFieldLabel("Image ID")
	UnknownCVECount                = newFieldLabel("Unknown CVE Count")
	FixableUnknownCVECount         = newFieldLabel("Fixable Unknown CVE Count")
	CriticalCVECount               = newFieldLabel("Critical CVE Count")
	FixableCriticalCVECount        = newFieldLabel("Fixable Critical CVE Count")
	ImportantCVECount              = newFieldLabel("Important CVE Count")
	FixableImportantCVECount       = newFieldLabel("Fixable Important CVE Count")
	ModerateCVECount               = newFieldLabel("Moderate CVE Count")
	FixableModerateCVECount        = newFieldLabel("Fixable Moderate CVE Count")
	LowCVECount                    = newFieldLabel("Low CVE Count")
	FixableLowCVECount             = newFieldLabel("Fixable Low CVE Count")

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
	PlatformComponent            = newFieldLabel("Platform Component")
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
	EntityType     = newFieldLabel("Entity Type")

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

	ComplianceOperatorIntegrationID            = newFieldLabel("Compliance Operator Integration ID")
	ComplianceOperatorVersion                  = newFieldLabel("Compliance Operator Version")
	ComplianceOperatorScanName                 = newFieldLabel("Compliance Scan Name")
	ComplianceOperatorInstalled                = newFieldLabel("Compliance Operator Installed")
	ComplianceOperatorSeverity                 = newFieldLabel("Compliance Rule Severity")
	ComplianceOperatorStatus                   = newFieldLabel("Compliance Operator Status")
	ComplianceOperatorCheckStatus              = newFieldLabel("Compliance Check Status")
	ComplianceOperatorRuleName                 = newFieldLabel("Compliance Rule Name")
	ComplianceOperatorProfileID                = newFieldLabel("Compliance Profile ID")
	ComplianceOperatorProfileName              = newFieldLabel("Compliance Profile Name")
	ComplianceOperatorConfigProfileName        = newFieldLabel("Compliance Config Profile Name")
	ComplianceOperatorProfileProductType       = newFieldLabel("Compliance Profile Product Type")
	ComplianceOperatorProfileVersion           = newFieldLabel("Compliance Profile Version")
	ComplianceOperatorStandard                 = newFieldLabel("Compliance Standard")
	ComplianceOperatorControl                  = newFieldLabel("Compliance Control")
	ComplianceOperatorScanConfig               = newFieldLabel("Compliance Scan Config ID")
	ComplianceOperatorScanConfigName           = newFieldLabel("Compliance Scan Config Name")
	ComplianceOperatorCheckID                  = newFieldLabel("Compliance Check ID")
	ComplianceOperatorCheckUID                 = newFieldLabel("Compliance Check UID")
	ComplianceOperatorCheckName                = newFieldLabel("Compliance Check Name")
	ComplianceOperatorCheckRationale           = newFieldLabel("Compliance Check Rationale")
	ComplianceOperatorCheckLastStartedTime     = newFieldLabel("Compliance Check Last Started Time")
	ComplianceOperatorScanUpdateTime           = newFieldLabel("Compliance Scan Config Last Updated Time")
	ComplianceOperatorResultCreateTime         = newFieldLabel("Compliance Check Result Created Time")
	ComplianceOperatorScanLastExecutedTime     = newFieldLabel("Compliance Scan Last Executed Time")
	ComplianceOperatorScanLastStartedTime      = newFieldLabel("Compliance Scan Last Started Time")
	ComplianceOperatorRuleType                 = newFieldLabel("Compliance Rule Type")
	ComplianceOperatorScanSettingBindingName   = newFieldLabel("Compliance Scan Setting Binding Name")
	ComplianceOperatorSuiteName                = newFieldLabel("Compliance Suite Name")
	ComplianceOperatorScanResult               = newFieldLabel("Compliance Scan Result")
	ComplianceOperatorProfileRef               = newFieldLabel("Profile Ref ID")
	ComplianceOperatorScanRef                  = newFieldLabel("Scan Ref ID")
	ComplianceOperatorRuleRef                  = newFieldLabel("Rule Ref ID")
	ComplianceOperatorRemediationName          = newFieldLabel("Compliance Remediation Name")
	ComplianceOperatorBenchmarkName            = newFieldLabel("Compliance Benchmark Name")
	ComplianceOperatorBenchmarkShortName       = newFieldLabel("Compliance Benchmark Short Name")
	ComplianceOperatorBenchmarkVersion         = newFieldLabel("Compliance Benchmark Version")
	ComplianceOperatorReportName               = newFieldLabel("Compliance Report Name")
	ComplianceOperatorReportState              = newFieldLabel("Compliance Report State")
	ComplianceOperatorReportStartedTime        = newFieldLabel("Compliance Report Started Time")
	ComplianceOperatorReportCompletedTime      = newFieldLabel("Compliance Report Completed Time")
	ComplianceOperatorReportRequestType        = newFieldLabel("Compliance Report Request Type")
	ComplianceOperatorReportNotificationMethod = newFieldLabel("Compliance Report Notification Method")

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
	ProfileCount    = newDerivedFieldLabel("Compliance Profile Name Count", ComplianceOperatorProfileName, CountDerivationType)
	// Translative derived fields with reversed sorting. These fields are supported only in pagination.
	NodePriority       = newDerivedFieldLabel("Node Risk Priority", NodeRiskScore, SimpleReverseSortDerivationType)
	DeploymentPriority = newDerivedFieldLabel("Deployment Risk Priority", DeploymentRiskScore, SimpleReverseSortDerivationType)
	ImagePriority      = newDerivedFieldLabel("Image Risk Priority", ImageRiskScore, SimpleReverseSortDerivationType)
	ComponentPriority  = newDerivedFieldLabel("Component Risk Priority", ComponentRiskScore, SimpleReverseSortDerivationType)

	// Custom derived fields to support query aliases.  These fields are only supported in pagination sort options.
	CompliancePassCount          = newDerivedFieldLabelWithType("Compliance Pass Count", ComplianceOperatorCheckStatus, CustomFieldType, postgres.Integer)
	ComplianceFailCount          = newDerivedFieldLabelWithType("Compliance Fail Count", ComplianceOperatorCheckStatus, CustomFieldType, postgres.Integer)
	ComplianceErrorCount         = newDerivedFieldLabelWithType("Compliance Error Count", ComplianceOperatorCheckStatus, CustomFieldType, postgres.Integer)
	ComplianceInfoCount          = newDerivedFieldLabelWithType("Compliance Info Count", ComplianceOperatorCheckStatus, CustomFieldType, postgres.Integer)
	ComplianceManualCount        = newDerivedFieldLabelWithType("Compliance Manual Count", ComplianceOperatorCheckStatus, CustomFieldType, postgres.Integer)
	ComplianceNotApplicableCount = newDerivedFieldLabelWithType("Compliance Not Applicable Count", ComplianceOperatorCheckStatus, CustomFieldType, postgres.Integer)
	ComplianceInconsistentCount  = newDerivedFieldLabelWithType("Compliance Inconsistent Count", ComplianceOperatorCheckStatus, CustomFieldType, postgres.Integer)

	// VM custom fields for sorting severity counts
	CriticalSeverityCount         = newDerivedFieldLabelWithType("Critical Severity Count", Severity, CustomFieldType, postgres.Integer)
	FixableCriticalSeverityCount  = newDerivedFieldLabelWithType("Fixable Critical Severity Count", Severity, CustomFieldType, postgres.Integer)
	ImportantSeverityCount        = newDerivedFieldLabelWithType("Important Severity Count", Severity, CustomFieldType, postgres.Integer)
	FixableImportantSeverityCount = newDerivedFieldLabelWithType("Fixable Important Severity Count", Severity, CustomFieldType, postgres.Integer)
	ModerateSeverityCount         = newDerivedFieldLabelWithType("Moderate Severity Count", Severity, CustomFieldType, postgres.Integer)
	FixableModerateSeverityCount  = newDerivedFieldLabelWithType("Fixable Moderate Severity Count", Severity, CustomFieldType, postgres.Integer)
	LowSeverityCount              = newDerivedFieldLabelWithType("Low Severity Count", Severity, CustomFieldType, postgres.Integer)
	FixableLowSeverityCount       = newDerivedFieldLabelWithType("Fixable Low Severity Count", Severity, CustomFieldType, postgres.Integer)
	UnknownSeverityCount          = newDerivedFieldLabelWithType("Unknown Severity Count", Severity, CustomFieldType, postgres.Integer)
	FixableUnknownSeverityCount   = newDerivedFieldLabelWithType("Fixable Unknown Severity Count", Severity, CustomFieldType, postgres.Integer)

	// Max-based derived fields.  These fields are primarily used in pagination.  If used in a select it will correspond
	// to the type of the reference field and simply provide the max function on that field.
	ComplianceLastScanMax            = newDerivedFieldLabel("Compliance Scan Last Executed Time Max", ComplianceOperatorScanLastExecutedTime, MaxDerivationType)
	SeverityMax                      = newDerivedFieldLabel("Severity Max", Severity, MaxDerivationType)
	CVSSMax                          = newDerivedFieldLabel("CVSS Max", CVSS, MaxDerivationType)
	CVECreatedTimeMin                = newDerivedFieldLabel("CVE Created Time Min", CVECreatedTime, MinDerivationType)
	EPSSProbablityMax                = newDerivedFieldLabel("EPSS Probability Max", EPSSProbablity, MaxDerivationType)
	ImpactScoreMax                   = newDerivedFieldLabel("Impact Score Max", ImpactScore, MaxDerivationType)
	FirstImageOccurrenceTimestampMin = newDerivedFieldLabel("First Image Occurrence Timestamp Min", FirstImageOccurrenceTimestamp, MinDerivationType)
	VulnerabilityStateMax            = newDerivedFieldLabel("Vulnerability State Max", VulnerabilityState, MaxDerivationType)
	NVDCVSSMax                       = newDerivedFieldLabel("NVD CVSS Max", NVDCVSS, MaxDerivationType)
	CVEPublishedOnMin                = newDerivedFieldLabel("CVE Published On Min", CVEPublishedOn, MinDerivationType)
	ComponentTopCVSSMax              = newDerivedFieldLabel("Component Top CVSS Max", ComponentTopCVSS, MaxDerivationType)
	// This is the priority which is essentially a reverse sort of the risk score
	ComponentPriorityMax = newDerivedFieldLabel("Component Risk Priority Score Max", ComponentRiskScore, MaxReverseSortDerivationType)

	// External network sources fields
	DefaultExternalSource    = newFieldLabel("Default External Source")
	DiscoveredExternalSource = newFieldLabel("Discovered External Source")
	ExternalSourceAddress    = newFieldLabel("External Source Address")

	// Report configurations search fields
	ReportName     = newFieldLabel("Report Name")
	ReportType     = newFieldLabel("Report Type")
	ReportConfigID = newFieldLabel("Report Configuration ID")
	// View Based report search fields
	AreaOfConcern = newFieldLabel("Area Of Concern")

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
	RequesterUserID             = newFieldLabel("Requester User ID")
	RequesterUserName           = newFieldLabel("Requester User Name")
	ApproverUserID              = newFieldLabel("Approver User ID")
	ApproverUserName            = newFieldLabel("Approver User Name")
	DeferralUpdateCVEs          = newFieldLabel("Deferral Update CVEs")
	FalsePositiveUpdateCVEs     = newFieldLabel("False Positive Update CVEs")

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

	// Integration fields.
	IntegrationID   = newFieldLabel("Integration ID")
	IntegrationName = newFieldLabel("Integration Name")
	IntegrationType = newFieldLabel("Integration Type")

	// AuthProvider fields.
	AuthProviderName = newFieldLabel("AuthProvider Name")

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
	TestEnum1Custom             = newDerivedFieldLabelWithType("Test String Affected By Enum1", TestEnum, CustomFieldType, postgres.Integer)
	TestEnum2Custom             = newDerivedFieldLabelWithType("Test String Affected By Enum2", TestEnum, CustomFieldType, postgres.Integer)
	TestInvalidEnumCustom       = newDerivedFieldLabelWithType("Invalid Test String Affected By Enum1", TestEnum, CustomFieldType, postgres.Integer)
)

func init() {
	derivedFields = set.NewStringSet()
	derivationsByField = make(map[string]map[string]DerivedTypeData)
	for k, metadata := range allFieldLabels {
		if metadata != nil {
			derivedFields.Add(strings.ToLower(k))
			derivedFromLower := strings.ToLower(string(metadata.DerivedFrom))
			subMap, exists := derivationsByField[derivedFromLower]
			if !exists {
				subMap = make(map[string]DerivedTypeData)
				derivationsByField[derivedFromLower] = subMap
			}
			subMap[k] = DerivedTypeData{
				DerivationType:  metadata.DerivationType,
				DerivedDataType: metadata.DerivedDataType,
			}
		}
	}
}

var (
	allFieldLabels     = make(map[string]*DerivedFieldLabelMetadata)
	derivationsByField map[string]map[string]DerivedTypeData
	derivedFields      set.StringSet
)

// IsValidFieldLabel returns whether this is a known, valid field label.
func IsValidFieldLabel(s string) bool {
	_, ok := allFieldLabels[strings.ToLower(s)]
	return ok
}

// GetFieldsDerivedFrom gets the fields derived from the given search field.
func GetFieldsDerivedFrom(s string) map[string]DerivedTypeData {
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

func newDerivedFieldLabelWithType(s string, derivedFrom FieldLabel, derivationType DerivationType, dataType postgres.DataType) FieldLabel {
	return newFieldLabelWithMetadata(s, &DerivedFieldLabelMetadata{
		DerivedFrom:     derivedFrom,
		DerivationType:  derivationType,
		DerivedDataType: dataType,
	})
}

func (f FieldLabel) String() string {
	return string(f)
}

func (f FieldLabel) Alias() string {
	return strings.ToLower(strings.Join(strings.Fields(string(f)), "_"))
}

// DerivedFieldLabelMetadata includes metadata showing that a field is derived.
type DerivedFieldLabelMetadata struct {
	DerivedFrom     FieldLabel
	DerivationType  DerivationType
	DerivedDataType postgres.DataType
}

// DerivedTypeData includes metadata showing that a field is derived.
type DerivedTypeData struct {
	DerivationType  DerivationType
	DerivedDataType postgres.DataType
}

// DerivationType represents a type of derivation.
//
//go:generate stringer -type=DerivationType
type DerivationType int

// This block enumerates all supported derivation types.
const (
	CountDerivationType DerivationType = iota
	SimpleReverseSortDerivationType
	MaxDerivationType
	CustomFieldType
	MinDerivationType
	MaxReverseSortDerivationType
)
