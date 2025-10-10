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
		ruleName := r.GetRuleName()
		rule := storage.ComplianceOperatorProfileV2_Rule_builder{
			RuleName: &ruleName,
		}.Build()
		rules = append(rules, rule)
	}

	productType := internalMsg.GetAnnotations()[v1alpha1.ProductTypeAnnotation]

	id := internalMsg.GetId()
	profileId := internalMsg.GetProfileId()
	name := internalMsg.GetName()
	profileVersion := internalMsg.GetProfileVersion()
	labels := internalMsg.GetLabels()
	annotations := internalMsg.GetAnnotations()
	description := internalMsg.GetDescription()
	product := internalMsg.GetAnnotations()[v1alpha1.ProductAnnotation]
	title := internalMsg.GetTitle()
	values := internalMsg.GetValues()
	profileRefId := BuildProfileRefID(clusterID, internalMsg.GetProfileId(), productType)

	return storage.ComplianceOperatorProfileV2_builder{
		Id:             &id,
		ProfileId:      &profileId,
		Name:           &name,
		ProfileVersion: &profileVersion,
		ProductType:    &productType,
		Labels:         labels,
		Annotations:    annotations,
		Description:    &description,
		Rules:          rules,
		Product:        &product,
		Title:          &title,
		Values:         values,
		ClusterId:      &clusterID,
		ProfileRefId:   &profileRefId,
	}.Build()
}
