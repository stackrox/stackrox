package storagetov2

import (
	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

func convertComplianceCheckStatus(status storage.ComplianceOperatorCheckResultV2_CheckStatus) v2.ComplianceCheckStatus {
	switch status {
	case storage.ComplianceOperatorCheckResultV2_PASS:
		return v2.ComplianceCheckStatus_PASS
	case storage.ComplianceOperatorCheckResultV2_FAIL:
		return v2.ComplianceCheckStatus_FAIL
	case storage.ComplianceOperatorCheckResultV2_ERROR:
		return v2.ComplianceCheckStatus_ERROR
	case storage.ComplianceOperatorCheckResultV2_INFO:
		return v2.ComplianceCheckStatus_INFO
	case storage.ComplianceOperatorCheckResultV2_MANUAL:
		return v2.ComplianceCheckStatus_MANUAL
	case storage.ComplianceOperatorCheckResultV2_NOT_APPLICABLE:
		return v2.ComplianceCheckStatus_NOT_APPLICABLE
	case storage.ComplianceOperatorCheckResultV2_INCONSISTENT:
		return v2.ComplianceCheckStatus_INCONSISTENT
	default:
		utils.Should(errors.Errorf("unhandled check result status encountered %s", status))
		return v2.ComplianceCheckStatus_MANUAL
	}
}
