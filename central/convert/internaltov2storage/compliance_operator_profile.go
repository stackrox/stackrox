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

	productType := internalMsg.GetAnnotations()[v1alpha1.ProductTypeAnnotation]

	profile := &storage.ComplianceOperatorProfileV2{
		Id:             internalMsg.GetId(),
		ProfileId:      internalMsg.GetProfileId(),
		Name:           internalMsg.GetName(),
		ProfileVersion: internalMsg.GetProfileVersion(),
		ProductType:    productType,
		Labels:         internalMsg.GetLabels(),
		Annotations:    internalMsg.GetAnnotations(),
		Description:    internalMsg.GetDescription(),
		Rules:          rules,
		Product:        internalMsg.GetAnnotations()[v1alpha1.ProductAnnotation],
		Title:          internalMsg.GetTitle(),
		Values:         internalMsg.GetValues(),
		ClusterId:      clusterID,
		ProfileRefId:   BuildProfileRefID(clusterID, internalMsg.GetProfileId(), productType),
		IsTailored:     internalMsg.GetIsTailored(),
	}

	// Convert tailored profile details if present
	if td := internalMsg.GetTailoredDetails(); td != nil {
		profile.TailoredDetails = convertTailoredProfileDetails(td)
	}

	return profile
}

// convertTailoredProfileDetails converts internal TailoredProfileDetails to storage format
func convertTailoredProfileDetails(td *central.TailoredProfileDetails) *storage.StorageTailoredProfileDetails {
	result := &storage.StorageTailoredProfileDetails{
		Extends:      td.GetExtends(),
		State:        td.GetState(),
		ErrorMessage: td.GetErrorMessage(),
	}

	for _, rule := range td.GetDisabledRules() {
		result.DisabledRules = append(result.DisabledRules, &storage.StorageTailoredProfileRuleModification{
			Name:      rule.GetName(),
			Rationale: rule.GetRationale(),
		})
	}

	for _, rule := range td.GetEnabledRules() {
		result.EnabledRules = append(result.EnabledRules, &storage.StorageTailoredProfileRuleModification{
			Name:      rule.GetName(),
			Rationale: rule.GetRationale(),
		})
	}

	for _, rule := range td.GetManualRules() {
		result.ManualRules = append(result.ManualRules, &storage.StorageTailoredProfileRuleModification{
			Name:      rule.GetName(),
			Rationale: rule.GetRationale(),
		})
	}

	for _, val := range td.GetSetValues() {
		result.SetValues = append(result.SetValues, &storage.StorageTailoredProfileValueOverride{
			Name:      val.GetName(),
			Value:     val.GetValue(),
			Rationale: val.GetRationale(),
		})
	}

	return result
}
