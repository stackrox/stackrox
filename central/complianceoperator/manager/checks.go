package manager

import (
	"fmt"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/complianceoperator/api/v1alpha1"
)

func statusToEvidence(result *storage.ComplianceOperatorCheckResult) (framework.Status, string) {
	switch result.GetStatus() {
	case storage.ComplianceOperatorCheckResult_PASS:
		return framework.PassStatus, fmt.Sprintf("Pass for %s", result.CheckName)
	case storage.ComplianceOperatorCheckResult_FAIL:
		return framework.FailStatus, fmt.Sprintf("Fail for %s", result.CheckName)
	case storage.ComplianceOperatorCheckResult_ERROR:
		return framework.FailStatus, fmt.Sprintf("Error for %s", result.CheckName)
	case storage.ComplianceOperatorCheckResult_INFO:
		return framework.SkipStatus, fmt.Sprintf("Skip for %s", result.CheckName)
	case storage.ComplianceOperatorCheckResult_MANUAL:
		return framework.SkipStatus, fmt.Sprintf("Manual for %s", result.CheckName)
	case storage.ComplianceOperatorCheckResult_NOT_APPLICABLE:
		return framework.SkipStatus, fmt.Sprintf("Not Applicable for %s", result.CheckName)
	case storage.ComplianceOperatorCheckResult_INCONSISTENT:
		return framework.FailStatus, fmt.Sprintf("Inconsistent for %s", result.CheckName)
	default:
		return framework.FailStatus, fmt.Sprintf("Unknown status for %s", result.CheckName)
	}
}

func platformCheckFunc(rule string) func(ctx framework.ComplianceContext) {
	return func(ctx framework.ComplianceContext) {
		results, ok := ctx.Data().ComplianceOperatorResults()[rule]
		if !ok {
			framework.Skipf(ctx, "Skipping check %v because no ComplianceCheckResults were found for it", rule)
			return
		}
		if len(results) != 1 {
			log.Errorf("UNEXPECTED RESULTS (%d) for platform check", len(results))
		}
		status, evidence := statusToEvidence(results[0])
		framework.RecordEvidence(ctx, status, evidence)
	}
}

func machineConfigCheckFunc(rule string) func(ctx framework.ComplianceContext) {
	return func(ctx framework.ComplianceContext) {
		rules := ctx.Data().ComplianceOperatorResults()[rule]
		framework.ForEachMachineConfig(ctx, func(ctx framework.ComplianceContext, machine string) {
			if len(rules) == 0 {
				framework.Skipf(ctx, "Skipping check %v because no ComplianceCheckResults were found for it", rule)
				return
			}
			for _, r := range rules {
				if r.Labels[v1alpha1.ComplianceScanLabel] == machine {
					status, evidence := statusToEvidence(r)
					framework.RecordEvidence(ctx, status, evidence)
					return
				}
			}
			framework.RecordEvidence(ctx, framework.FailStatus, fmt.Sprintf("No result found for machine config: %q", machine))
		})
	}
}
