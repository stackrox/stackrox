package complianceoperator

// GroupVersionResources for compliance operator resources
var (
	CheckResultGVR        = GetGroupVersion().WithResource("compliancecheckresults")
	ProfileGVR            = GetGroupVersion().WithResource("profiles")
	TailoredProfileGVR    = GetGroupVersion().WithResource("tailoredprofiles")
	ScanSettingGVR        = GetGroupVersion().WithResource("scansettings")
	ScanSettingBindingGVR = GetGroupVersion().WithResource("scansettingbindings")
	ScanGVR               = GetGroupVersion().WithResource("compliancescans")
	RuleGVR               = GetGroupVersion().WithResource("rules")
)
