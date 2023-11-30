package internaltostorage

import (
	"fmt"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

const (
	ocpComplianceLabelsKey = "compliance.openshift.io/"
)

var (
	productKey     = ocpComplianceLabelsKey + "product"
	productTypeKey = ocpComplianceLabelsKey + "product-type"
)

// ComplianceOperatorProfileV2 converts internal api profiles to V2 storage profiles
func ComplianceOperatorProfileV2(internalMsg *central.ComplianceOperatorProfileV2) *storage.ComplianceOperatorProfileV2 {
	// The primary key is name-version if version is present.  Just name if it is not.
	key := internalMsg.GetName()
	if internalMsg.GetProfileVersion() != "" {
		key = fmt.Sprintf("%s-%s", key, internalMsg.GetProfileVersion())
	}

	var rules []*storage.ComplianceOperatorProfileV2_Rule
	for _, r := range internalMsg.GetRules() {
		rules = append(rules, &storage.ComplianceOperatorProfileV2_Rule{
			RuleName: r.GetRuleName(),
		})
	}

	return &storage.ComplianceOperatorProfileV2{
		Id:             key,
		ProfileId:      internalMsg.GetProfileId(),
		Name:           internalMsg.GetName(),
		ProfileVersion: internalMsg.GetProfileVersion(),
		ProductType:    internalMsg.GetAnnotations()[productTypeKey],
		Labels:         internalMsg.GetLabels(),
		Annotations:    internalMsg.GetAnnotations(),
		Description:    internalMsg.GetDescription(),
		Rules:          rules,
		Product:        internalMsg.GetAnnotations()[productKey],
		Title:          internalMsg.GetTitle(),
		Values:         internalMsg.GetValues(),
	}
}

// ComplianceOperatorProfileV1 converts V2 internal api profiles to V1 storage profiles
func ComplianceOperatorProfileV1(internalMsg *central.ComplianceOperatorProfileV2, clusterID string) *storage.ComplianceOperatorProfile {
	var rules []*storage.ComplianceOperatorProfile_Rule
	for _, r := range internalMsg.GetRules() {
		rules = append(rules, &storage.ComplianceOperatorProfile_Rule{
			Name: r.GetRuleName(),
		})
	}

	return &storage.ComplianceOperatorProfile{
		Id:          internalMsg.GetId(),
		ProfileId:   internalMsg.GetProfileId(),
		Name:        internalMsg.GetName(),
		ClusterId:   clusterID,
		Labels:      internalMsg.GetLabels(),
		Annotations: internalMsg.GetAnnotations(),
		Description: internalMsg.GetDescription(),
		Rules:       rules,
	}
}

func convertSeverity(severity central.ComplianceOperatorRuleSeverity) storage.RuleSeverity {
	switch severity {
	case central.ComplianceOperatorRuleSeverity_HIGH_RULE_SEVERITY:
		return storage.RuleSeverity_HIGH_RULE_SEVERITY
	case central.ComplianceOperatorRuleSeverity_MEDIUM_RULE_SEVERITY:
		return storage.RuleSeverity_MEDIUM_RULE_SEVERITY
	case central.ComplianceOperatorRuleSeverity_LOW_RULE_SEVERITY:
		return storage.RuleSeverity_LOW_RULE_SEVERITY
	case central.ComplianceOperatorRuleSeverity_INFO_RULE_SEVERITY:
		return storage.RuleSeverity_INFO_RULE_SEVERITY
	case central.ComplianceOperatorRuleSeverity_UNKNOWN_RULE_SEVERITY:
		return storage.RuleSeverity_UNKNOWN_RULE_SEVERITY
	default:
		return storage.RuleSeverity_UNSET_RULE_SEVERITY
	}
}
