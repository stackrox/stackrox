package complianceoperator

import (
	compv1alpha1 "github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// Compliance operator API set.
	groupVersion = compv1alpha1.SchemeGroupVersion

	// List of required compliance operator CRDs.
	requiredGVKs = make([]schema.GroupVersionKind, 0)
)

// GroupVersionKind required for compliance operator custom resources.
var (
	ProfileGVK               = registerGVK("Profile")
	RuleGVK                  = registerGVK("Rule")
	ScanSettingGVK           = registerGVK("ScanSetting")
	ScanSettingBindingGVK    = registerGVK("ScanSetting")
	ComplianceSuiteGVK       = registerGVK("ComplianceSuite")
	ComplianceScanGVK        = registerGVK("ComplianceScan")
	ComplianceCheckResultGVK = registerGVK("ComplianceCheckResult")
)

// GetGroupVersion return the group version that uniquely represents the APIs to for compliance operator CRs.
func GetGroupVersion() schema.GroupVersion {
	return groupVersion
}

// GetAllRequiredGVKs returns an array of GVK required by ACS compliance workflows.
func GetAllRequiredGVKs() []schema.GroupVersionKind {
	return requiredGVKs
}

func registerGVK(kind string) schema.GroupVersionKind {
	gvk := GetGroupVersion().WithKind(kind)
	requiredGVKs = append(requiredGVKs, gvk)
	return gvk
}
