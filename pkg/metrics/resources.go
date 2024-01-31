package metrics

// Resource represents the resource that we want to time.
//
//go:generate stringer -type=Resource
type Resource int

// The following is the list of Resources that we want to time.
const (
	Alert Resource = iota
	Deployment
	ProcessIndicator
	ProcessListeningOnPort
	Image
	Secret
	Namespace
	NetworkPolicy
	Node
	NodeInventory
	ProviderMetadata
	ComplianceReturn
	ImageIntegration
	ServiceAccount
	PermissionSet
	Role
	RoleBinding
	DeploymentReprocess
	Pod
	ComplianceOperatorCheckResult
	ComplianceOperatorProfile
	ComplianceOperatorScanSettingBinding
	ComplianceOperatorRule
	ComplianceOperatorScan
	ComplianceOperatorInfo
	ComplianceOperatorCheckResultV2
	ComplianceOperatorRuleV2
	ComplianceOperatorProfileV2
	ComplianceOperatorScanV2
	ComplianceOperatorSuite
)
