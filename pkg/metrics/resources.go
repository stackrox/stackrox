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
	Image
	Secret
	Namespace
	NetworkPolicy
	Node
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
)
