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

	return &storage.ComplianceOperatorProfileV2{
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
		OperatorKind:   centralToStorageProfileKind(internalMsg.GetOperatorKind()),
	}
}

func centralToStorageProfileKind(kind central.ComplianceOperatorProfileV2_OperatorKind) storage.ComplianceOperatorProfileV2_OperatorKind {
	switch kind {
	case central.ComplianceOperatorProfileV2_PROFILE:
		return storage.ComplianceOperatorProfileV2_PROFILE
	case central.ComplianceOperatorProfileV2_TAILORED_PROFILE:
		return storage.ComplianceOperatorProfileV2_TAILORED_PROFILE
	case central.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED:
		// ROX-31229: Older sensors do not set OperatorKind for regular (non-tailored)
		// profiles, so UNSPECIFIED is treated as PROFILE. This fallback can be
		// removed once versions that don't set OperatorKind (<= 4.10) are not supported.
		return storage.ComplianceOperatorProfileV2_PROFILE
	default:
		log.Warnf("Unexpected profile operator kind %v", kind)
		return storage.ComplianceOperatorProfileV2_OPERATOR_KIND_UNSPECIFIED
	}
}
