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

// DeriveAggregateDataFreshness computes the aggregate freshness over a set of
// check results from COUNTs of their stored (check-level) freshness: how many
// results are STALE and how many are CURRENT. Counting (rather than taking the
// MIN/MAX of the stored enum) is required because UNKNOWN sorts between CURRENT
// and STALE: a group containing CURRENT, STALE, and UNKNOWN rows would have
// MIN==UNKNOWN, which would hide the genuine CURRENT/STALE mix. Counts are
// unaffected by how many UNKNOWN rows are present, since UNKNOWN rows are not
// counted by either staleCount or currentCount.
//
// INCONSISTENT means the set mixes stale and current data, which points at a
// partial problem (some results arrived, some are stale) rather than a
// whole-set problem (all stale). This is the single source of truth for
// deriving aggregate freshness: every read path (profile/cluster stats
// dashboards, per-cluster check result rollups) must call this function.
func DeriveAggregateDataFreshness(staleCount, currentCount int) v2.ComplianceAggregateDataFreshness {
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
