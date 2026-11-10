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

func convertComplianceDataFreshness(freshness storage.ComplianceOperatorCheckResultV2_DataFreshness) v2.ComplianceCheckDataFreshness {
	switch freshness {
	case storage.ComplianceOperatorCheckResultV2_UNKNOWN:
		return v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_UNKNOWN
	case storage.ComplianceOperatorCheckResultV2_CURRENT:
		return v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT
	case storage.ComplianceOperatorCheckResultV2_STALE:
		return v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE
	default:
		utils.Should(errors.Errorf("unhandled data freshness encountered %s", freshness))
		return v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_UNKNOWN
	}
}

// aggregateDataFreshness rolls up per-check freshness counts into the v2 aggregate
// enum: any stale with any current is inconsistent, otherwise it reflects whichever
// of stale/current is present, else unknown.
func aggregateDataFreshness(currentCount, staleCount int) v2.ComplianceAggregateDataFreshness {
	switch {
	case staleCount > 0 && currentCount > 0:
		return v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_INCONSISTENT
	case staleCount > 0:
		return v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_STALE
	case currentCount > 0:
		return v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_CURRENT
	default:
		return v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_UNKNOWN
	}
}
