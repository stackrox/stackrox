package fixtures

import (
	"math/rand"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetComplianceCheckResult returns a test compliance check result
func GetComplianceCheckResult(name, clusterID, clusterName, scanName, scanConfigName, scanRefID string) *storage.ComplianceOperatorCheckResultV2 {
	cocrv2 := &storage.ComplianceOperatorCheckResultV2{}
	cocrv2.SetId(uuid.NewV4().String())
	cocrv2.SetCheckId(name)
	cocrv2.SetCheckName(name)
	cocrv2.SetClusterId(clusterID)
	cocrv2.SetStatus(storage.ComplianceOperatorCheckResultV2_CheckStatus(rand.Intn(7) + 1))
	cocrv2.SetSeverity(0)
	cocrv2.SetDescription("test description " + name)
	cocrv2.SetInstructions("test instruction " + name)
	cocrv2.SetLabels(nil)
	cocrv2.SetAnnotations(nil)
	cocrv2.ClearCreatedTime()
	cocrv2.SetValuesUsed(nil)
	cocrv2.SetWarnings(nil)
	cocrv2.SetScanName(scanName)
	cocrv2.SetClusterName(clusterName)
	cocrv2.SetScanConfigName(scanConfigName)
	cocrv2.SetRationale("test rationale " + name)
	cocrv2.SetScanRefId(scanRefID)
	return cocrv2
}

// GetExpectedBenchmark returns a test benchmark
func GetExpectedBenchmark() []*storage.ComplianceOperatorBenchmarkV2 {
	return []*storage.ComplianceOperatorBenchmarkV2{
		storage.ComplianceOperatorBenchmarkV2_builder{
			Id:          uuid.NewV4().String(),
			Name:        "CIS Benchmark",
			Version:     "1.5",
			Description: "blah",
			Provider:    "",
			ShortName:   "OCP_CIS",
			Profiles: []*storage.ComplianceOperatorBenchmarkV2_Profile{
				storage.ComplianceOperatorBenchmarkV2_Profile_builder{ProfileName: "ocp4", ProfileVersion: "1.5"}.Build(),
			},
		}.Build(),
	}
}
