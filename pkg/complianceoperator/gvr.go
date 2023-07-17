package complianceoperator

// GroupVersionResources for compliance operator resources
var (
	ComplianceCheckResultGVR = GetGroupVersion().WithResource("compliancecheckresults")
	ProfileGVR               = GetGroupVersion().WithResource("profiles")
	TailoredProfileGVR       = GetGroupVersion().WithResource("tailoredprofiles")
	ScanSettingGVR           = GetGroupVersion().WithResource("scansettings")
	ScanSettingBindingGVR    = GetGroupVersion().WithResource("scansettingbindings")
	ComplianceScanGVR        = GetGroupVersion().WithResource("compliancescans")
	RuleGVR                  = GetGroupVersion().WithResource("rules")
)
