package storagetov2

import (
	"github.com/pkg/errors"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/utils"
)

// convertComplianceCheckDataFreshness converts the stored per-check freshness to the
// v2 check-level freshness. This is a straight passthrough since a single check result
// can never be INCONSISTENT.
func convertComplianceCheckDataFreshness(freshness storage.ComplianceDataFreshness) v2.ComplianceCheckDataFreshness {
	switch freshness {
	case storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_CURRENT:
		return v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_CURRENT
	case storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_STALE:
		return v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_STALE
	case storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_UNKNOWN:
		return v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_UNKNOWN
	default:
		utils.Should(errors.Errorf("unhandled compliance data freshness encountered %s", freshness))
		return v2.ComplianceCheckDataFreshness_COMPLIANCE_CHECK_DATA_FRESHNESS_UNKNOWN
	}
}

// deriveAggregateDataFreshness computes the aggregate freshness over a set of
// check results from the MIN and MAX of their stored (check-level) freshness.
// INCONSISTENT means the set mixes stale and non-stale (current) data, which
// points at a partial problem (some results arrived, some are stale) rather than
// a whole-set problem (all stale). UNKNOWN rows (not yet evaluated) do not by
// themselves make an otherwise-uniform set inconsistent.
func deriveAggregateDataFreshness(minF, maxF storage.ComplianceDataFreshness) v2.ComplianceAggregateDataFreshness {
	// A genuine mix of CURRENT and STALE present together.
	if maxF == storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_STALE &&
		minF == storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_CURRENT {
		return v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_INCONSISTENT
	}
	switch maxF {
	case storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_STALE:
		return v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_STALE
	case storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_CURRENT:
		return v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_CURRENT
	case storage.ComplianceDataFreshness_COMPLIANCE_DATA_FRESHNESS_UNKNOWN:
		return v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_UNKNOWN
	default:
		utils.Should(errors.Errorf("unhandled compliance data freshness encountered %s", maxF))
		return v2.ComplianceAggregateDataFreshness_COMPLIANCE_AGGREGATE_DATA_FRESHNESS_UNKNOWN
	}
}

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
