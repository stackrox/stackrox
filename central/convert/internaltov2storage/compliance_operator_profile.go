package internaltov2storage

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

// ComplianceOperatorProfileV2 converts internal api profiles to V2 storage profiles
func ComplianceOperatorProfileV2(internalMsg *central.ComplianceOperatorProfileV2, clusterID string) *storage.ComplianceOperatorProfileV2 {
	var rules []*storage.ComplianceOperatorProfileV2_Rule
	for _, r := range internalMsg.GetRules() {
		cr := &storage.ComplianceOperatorProfileV2_Rule{}
		cr.SetRuleName(r.GetRuleName())
		rules = append(rules, cr)
	}

	productType := internalMsg.GetAnnotations()[v1alpha1.ProductTypeAnnotation]

	copv2 := &storage.ComplianceOperatorProfileV2{}
	copv2.SetId(internalMsg.GetId())
	copv2.SetProfileId(internalMsg.GetProfileId())
	copv2.SetName(internalMsg.GetName())
	copv2.SetProfileVersion(internalMsg.GetProfileVersion())
	copv2.SetProductType(productType)
	copv2.SetLabels(internalMsg.GetLabels())
	copv2.SetAnnotations(internalMsg.GetAnnotations())
	copv2.SetDescription(internalMsg.GetDescription())
	copv2.SetRules(rules)
	copv2.SetProduct(internalMsg.GetAnnotations()[v1alpha1.ProductAnnotation])
	copv2.SetTitle(internalMsg.GetTitle())
	copv2.SetValues(internalMsg.GetValues())
	copv2.SetClusterId(clusterID)
	copv2.SetProfileRefId(BuildProfileRefID(clusterID, internalMsg.GetProfileId(), productType))
	return copv2
}
