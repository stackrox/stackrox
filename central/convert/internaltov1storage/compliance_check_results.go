package internaltov1storage

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
)

var (
	internalStatusToV1 = map[central.ComplianceOperatorCheckResultV2_CheckStatus]storage.ComplianceOperatorCheckResult_CheckStatus{
		central.ComplianceOperatorCheckResultV2_UNSET:          storage.ComplianceOperatorCheckResult_UNSET,
		central.ComplianceOperatorCheckResultV2_PASS:           storage.ComplianceOperatorCheckResult_PASS,
		central.ComplianceOperatorCheckResultV2_FAIL:           storage.ComplianceOperatorCheckResult_FAIL,
		central.ComplianceOperatorCheckResultV2_ERROR:          storage.ComplianceOperatorCheckResult_ERROR,
		central.ComplianceOperatorCheckResultV2_INFO:           storage.ComplianceOperatorCheckResult_INFO,
		central.ComplianceOperatorCheckResultV2_MANUAL:         storage.ComplianceOperatorCheckResult_MANUAL,
		central.ComplianceOperatorCheckResultV2_NOT_APPLICABLE: storage.ComplianceOperatorCheckResult_NOT_APPLICABLE,
		central.ComplianceOperatorCheckResultV2_INCONSISTENT:   storage.ComplianceOperatorCheckResult_INCONSISTENT,
	}
)

// ConvertInternalToV1Storage converts internal api V2 check result to a V1 storage check result
func ConvertInternalToV1Storage(internalResult *central.ComplianceOperatorCheckResultV2) *storage.ComplianceOperatorCheckResult {
	return &storage.ComplianceOperatorCheckResult{
		Id:           internalResult.GetId(),
		CheckId:      internalResult.GetCheckId(),
		CheckName:    internalResult.GetCheckName(),
		ClusterId:    internalResult.GetClusterId(),
		Status:       internalStatusToV1[internalResult.GetStatus()],
		Description:  internalResult.GetDescription(),
		Instructions: internalResult.GetInstructions(),
		Labels:       internalResult.GetLabels(),
		Annotations:  internalResult.GetAnnotations(),
	}
}
