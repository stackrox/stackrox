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
		rules = append(rules, &storage.ComplianceOperatorProfileV2_Rule{
			RuleName: r.GetRuleName(),
		})
	}

	return &storage.ComplianceOperatorProfileV2{
		Id:             internalMsg.GetId(),
		ProfileId:      internalMsg.GetProfileId(),
		Name:           internalMsg.GetName(),
		ProfileVersion: internalMsg.GetProfileVersion(),
		ProductType:    internalMsg.GetAnnotations()[v1alpha1.ProductTypeAnnotation],
		Labels:         internalMsg.GetLabels(),
		Annotations:    internalMsg.GetAnnotations(),
		Description:    internalMsg.GetDescription(),
		Rules:          rules,
		Product:        internalMsg.GetAnnotations()[v1alpha1.ProductAnnotation],
		Title:          internalMsg.GetTitle(),
		Values:         internalMsg.GetValues(),
		ClusterId:      clusterID,
	}
}
