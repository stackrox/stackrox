package internaltov2storage

import (
	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

const (
	LastScannedAnnotationKey = "compliance.openshift.io/last-scanned-timestamp"
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
func ComplianceOperatorCheckResult(sensorData *central.ComplianceOperatorCheckResultV2, clusterID string, clusterName string) *storage.ComplianceOperatorCheckResultV2 {
	lastStartedTimestamp, _ := protocompat.ParseRFC3339NanoTimestamp(sensorData.GetAnnotations()[LastScannedAnnotationKey])

	id := sensorData.GetId()
	checkId := sensorData.GetCheckId()
	checkName := sensorData.GetCheckName()
	status := statusToV2[sensorData.GetStatus()]
	severity := severityToV2[sensorData.GetSeverity()]
	description := sensorData.GetDescription()
	instructions := sensorData.GetInstructions()
	labels := sensorData.GetLabels()
	annotations := sensorData.GetAnnotations()
	createdTime := sensorData.GetCreatedTime()
	scanName := sensorData.GetScanName()
	scanConfigName := sensorData.GetSuiteName()
	rationale := sensorData.GetRationale()
	valuesUsed := sensorData.GetValuesUsed()
	warnings := sensorData.GetWarnings()
	scanRefId := BuildNameRefID(clusterID, sensorData.GetScanName())
	ruleRefId := BuildNameRefID(clusterID, sensorData.GetAnnotations()[v1alpha1.RuleIDAnnotationKey])

	return storage.ComplianceOperatorCheckResultV2_builder{
		Id:              &id,
		CheckId:         &checkId,
		CheckName:       &checkName,
		ClusterId:       &clusterID,
		ClusterName:     &clusterName,
		Status:          &status,
		Severity:        &severity,
		Description:     &description,
		Instructions:    &instructions,
		Labels:          labels,
		Annotations:     annotations,
		CreatedTime:     createdTime,
		ScanName:        &scanName,
		ScanConfigName:  &scanConfigName,
		Rationale:       &rationale,
		ValuesUsed:      valuesUsed,
		Warnings:        warnings,
		ScanRefId:       &scanRefId,
		RuleRefId:       &ruleRefId,
		LastStartedTime: lastStartedTimestamp,
	}.Build()
}
