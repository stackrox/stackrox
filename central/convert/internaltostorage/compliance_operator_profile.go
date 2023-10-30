package internaltostorage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

func ComplianceOperatorProfileV2(internalMsg *central.ComplianceOperatorProfileV2) *storage.ComplianceOperatorProfileV2 {
	return &storage.ComplianceOperatorProfileV2{
		Id:              internalMsg.GetId(),
		ProfileId:       internalMsg.GetProfileId(),
		Name:            internalMsg.GetName(),
		ProfileVersion:  internalMsg.GetProfileVersion(),
		OperatorVersion: "",  // TODO figured out how to get this
		ProductType:     nil, // TODO pull from labels
		Standard:        "",  // TODO pull from labels
		Labels:          internalMsg.GetLabels(),
		Annotations:     internalMsg.GetAnnotations(),
		Description:     internalMsg.GetDescription(),
		// Rules: internalMsg.GetRules(), // TODO figure out how to do these
		Product: "", // TODO:  pull from labels and annotations

	}
}

func ComplianceOperatorProfileV1(internalMsg *central.ComplianceOperatorProfileV2, clusterID string) *storage.ComplianceOperatorProfile {
	return &storage.ComplianceOperatorProfile{
		Id:          internalMsg.GetId(),
		ProfileId:   internalMsg.GetProfileId(),
		Name:        internalMsg.GetName(),
		ClusterId:   clusterID,
		Labels:      internalMsg.GetLabels(),
		Annotations: internalMsg.GetAnnotations(),
		Description: internalMsg.GetDescription(),
		//Rules: internalMsg.GetRules(), // TODO figure out how to do these
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
