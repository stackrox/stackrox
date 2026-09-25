package storagetov2

import (
	"testing"
	"time"

	compliancedata "github.com/stackrox/rox/central/complianceoperator/v2/compliancedata"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestComplianceV2SpecificCheckResult_DataState verifies the per-cluster data
// state is computed from the resolver (regression for the converter previously
// hardcoding UNKNOWN, so GetComplianceProfileCheckDetails never reported state).
func TestComplianceV2SpecificCheckResult_DataState(t *testing.T) {
	// Daily 02:00 UTC. now = 15:00, grace 85m ⇒ reference fire = 02:00 today.
	now := time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC)
	cfg := &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "daily",
		Schedule:       &storage.Schedule{IntervalType: storage.Schedule_DAILY, Hour: 2, Minute: 0},
	}
	resolver := compliancedata.NewConfigResolver([]*storage.ComplianceOperatorScanConfigurationV2{cfg}, now)

	results := []*storage.ComplianceOperatorCheckResultV2{
		{
			CheckId: "cid", CheckName: "check1", ClusterId: "cluster-current",
			Status:          storage.ComplianceOperatorCheckResultV2_PASS,
			ScanConfigName:  "daily",
			LastStartedTime: timestamppb.New(time.Date(2026, 9, 2, 2, 5, 0, 0, time.UTC)), // after fire
		},
		{
			CheckId: "cid", CheckName: "check1", ClusterId: "cluster-outdated",
			Status:          storage.ComplianceOperatorCheckResultV2_PASS,
			ScanConfigName:  "daily",
			LastStartedTime: timestamppb.New(time.Date(2026, 9, 1, 2, 0, 0, 0, time.UTC)), // before fire
		},
	}

	converted := ComplianceV2SpecificCheckResult(results, "check1", nil, resolver)
	if assert.NotNil(t, converted) && assert.Len(t, converted.GetClusters(), 2) {
		byCluster := map[string]v2.ComplianceDataState{}
		for _, c := range converted.GetClusters() {
			byCluster[c.GetCluster().GetClusterId()] = c.GetDataState()
		}
		assert.Equal(t, v2.ComplianceDataState_COMPLIANCE_DATA_STATE_CURRENT, byCluster["cluster-current"])
		assert.Equal(t, v2.ComplianceDataState_COMPLIANCE_DATA_STATE_OUTDATED, byCluster["cluster-outdated"])
	}

	// nil resolver ⇒ UNKNOWN (backwards-compatible default).
	convertedNil := ComplianceV2SpecificCheckResult(results, "check1", nil, nil)
	if assert.NotNil(t, convertedNil) {
		for _, c := range convertedNil.GetClusters() {
			assert.Equal(t, v2.ComplianceDataState_COMPLIANCE_DATA_STATE_UNKNOWN, c.GetDataState())
		}
	}
}

// TestComplianceV2CheckResults_DataState verifies the single-cluster Coverage
// view converter sets the per-check "Data status" from the resolver.
func TestComplianceV2CheckResults_DataState(t *testing.T) {
	now := time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC)
	cfg := &storage.ComplianceOperatorScanConfigurationV2{
		ScanConfigName: "daily",
		Schedule:       &storage.Schedule{IntervalType: storage.Schedule_DAILY, Hour: 2, Minute: 0},
	}
	resolver := compliancedata.NewConfigResolver([]*storage.ComplianceOperatorScanConfigurationV2{cfg}, now)

	results := []*storage.ComplianceOperatorCheckResultV2{
		{
			CheckName: "check-current", RuleRefId: "ref", ScanConfigName: "daily",
			Status:          storage.ComplianceOperatorCheckResultV2_PASS,
			LastStartedTime: timestamppb.New(time.Date(2026, 9, 2, 2, 5, 0, 0, time.UTC)), // after fire
		},
		{
			CheckName: "check-outdated", RuleRefId: "ref", ScanConfigName: "daily",
			Status:          storage.ComplianceOperatorCheckResultV2_PASS,
			LastStartedTime: timestamppb.New(time.Date(2026, 9, 1, 2, 0, 0, 0, time.UTC)), // before fire
		},
	}

	converted := ComplianceV2CheckResults(results, map[string]string{"ref": "rule"}, nil, resolver)
	if assert.Len(t, converted, 2) {
		byName := map[string]v2.ComplianceDataState{}
		for _, r := range converted {
			byName[r.GetCheckName()] = r.GetDataState()
		}
		assert.Equal(t, v2.ComplianceDataState_COMPLIANCE_DATA_STATE_CURRENT, byName["check-current"])
		assert.Equal(t, v2.ComplianceDataState_COMPLIANCE_DATA_STATE_OUTDATED, byName["check-outdated"])
	}

	// nil resolver ⇒ UNKNOWN default.
	for _, r := range ComplianceV2CheckResults(results, map[string]string{"ref": "rule"}, nil, nil) {
		assert.Equal(t, v2.ComplianceDataState_COMPLIANCE_DATA_STATE_UNKNOWN, r.GetDataState())
	}
}
