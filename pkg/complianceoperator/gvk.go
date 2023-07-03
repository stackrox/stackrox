package complianceoperator

import (
	compv1alpha1 "github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	requiredObjectTypes = []runtime.Object{
		&compv1alpha1.Profile{},
		&compv1alpha1.Rule{},
		&compv1alpha1.ScanSetting{},
		&compv1alpha1.ScanSettingBinding{},
		&compv1alpha1.ComplianceScan{},
		&compv1alpha1.ComplianceSuite{},
		&compv1alpha1.ComplianceCheckResult{},
	}
)

// GetAllRequiredComplianceGVKs returns an array of GVK required by ACS compliance workflows.
func GetAllRequiredComplianceGVKs() []schema.GroupVersionKind {
	gvk := make([]schema.GroupVersionKind, 0, len(requiredObjectTypes))
	for _, obj := range requiredObjectTypes {
		gvk = append(gvk, obj.GetObjectKind().GroupVersionKind())
	}
	return gvk
}
