package internaltov2storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

var (
	statusToV2 = map[central.ComplianceOperatorCheckResultV2_CheckStatus]storage.ComplianceOperatorCheckResultV2_CheckStatus{
		central.ComplianceOperatorCheckResultV2_UNSET:          storage.ComplianceOperatorCheckResultV2_UNSET,
		central.ComplianceOperatorCheckResultV2_PASS:           storage.ComplianceOperatorCheckResultV2_PASS,
		central.ComplianceOperatorCheckResultV2_FAIL:           storage.ComplianceOperatorCheckResultV2_FAIL,
		central.ComplianceOperatorCheckResultV2_ERROR:          storage.ComplianceOperatorCheckResultV2_ERROR,
		central.ComplianceOperatorCheckResultV2_INFO:           storage.ComplianceOperatorCheckResultV2_INFO,
		central.ComplianceOperatorCheckResultV2_MANUAL:         storage.ComplianceOperatorCheckResultV2_MANUAL,
		central.ComplianceOperatorCheckResultV2_NOT_APPLICABLE: storage.ComplianceOperatorCheckResultV2_NOT_APPLICABLE,
		central.ComplianceOperatorCheckResultV2_INCONSISTENT:   storage.ComplianceOperatorCheckResultV2_INCONSISTENT,
	}

	severityToV2 = map[central.ComplianceOperatorRuleSeverity]storage.RuleSeverity{
		central.ComplianceOperatorRuleSeverity_UNSET_RULE_SEVERITY:   storage.RuleSeverity_UNSET_RULE_SEVERITY,
		central.ComplianceOperatorRuleSeverity_UNKNOWN_RULE_SEVERITY: storage.RuleSeverity_UNKNOWN_RULE_SEVERITY,
		central.ComplianceOperatorRuleSeverity_INFO_RULE_SEVERITY:    storage.RuleSeverity_INFO_RULE_SEVERITY,
		central.ComplianceOperatorRuleSeverity_LOW_RULE_SEVERITY:     storage.RuleSeverity_LOW_RULE_SEVERITY,
		central.ComplianceOperatorRuleSeverity_MEDIUM_RULE_SEVERITY:  storage.RuleSeverity_MEDIUM_RULE_SEVERITY,
		central.ComplianceOperatorRuleSeverity_HIGH_RULE_SEVERITY:    storage.RuleSeverity_HIGH_RULE_SEVERITY,
	}
)

// ComplianceOperatorCheckResult converts internal api V2 check result to a V2 storage check result
func ComplianceOperatorCheckResult(sensorData *central.ComplianceOperatorCheckResultV2, clusterID string) *storage.ComplianceOperatorCheckResultV2 {
	return &storage.ComplianceOperatorCheckResultV2{
		Id:             sensorData.GetId(),
		CheckId:        sensorData.GetCheckId(),
		CheckName:      sensorData.GetCheckName(),
		ClusterId:      clusterID,
		Status:         statusToV2[sensorData.GetStatus()],
		Severity:       severityToV2[sensorData.GetSeverity()],
		Description:    sensorData.GetDescription(),
		Instructions:   sensorData.GetInstructions(),
		Labels:         sensorData.GetLabels(),
		Annotations:    sensorData.GetAnnotations(),
		CreatedTime:    sensorData.GetCreatedTime(),
		ScanConfigName: sensorData.GetSuiteName(),
		Rationale:      sensorData.GetRationale(),
		ValuesUsed:     sensorData.GetValuesUsed(),
		Warnings:       sensorData.GetWarnings(),
	}
}
