package manager

import (
	"fmt"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

func formatEvidence(status string, result *storage.ComplianceOperatorCheckResult) string {
	return fmt.Sprintf("%s for %s. Please see the Compliance Operator's full report in ARF format for more detailed evidence.", status, result.GetCheckName())
}

func statusToEvidence(result *storage.ComplianceOperatorCheckResult) (framework.Status, string) {
	switch result.GetStatus() {
	case storage.ComplianceOperatorCheckResult_PASS:
		return framework.PassStatus, formatEvidence("Pass", result)
	case storage.ComplianceOperatorCheckResult_FAIL:
		return framework.FailStatus, formatEvidence("Fail", result)
	case storage.ComplianceOperatorCheckResult_ERROR:
		return framework.FailStatus, formatEvidence("Error", result)
	case storage.ComplianceOperatorCheckResult_INFO:
		return framework.SkipStatus, formatEvidence("Skip", result)
	case storage.ComplianceOperatorCheckResult_MANUAL:
		return framework.SkipStatus, formatEvidence("Manual", result)
	case storage.ComplianceOperatorCheckResult_NOT_APPLICABLE:
		return framework.SkipStatus, formatEvidence("Not Applicable", result)
	case storage.ComplianceOperatorCheckResult_INCONSISTENT:
		return framework.FailStatus, formatEvidence("Inconsistent", result)
	default:
		return framework.FailStatus, formatEvidence("Unknown status", result)
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
			framework.RecordEvidence(ctx, framework.InternalSkipStatus, fmt.Sprintf("No result found for machine config: %q", machine))
		})
	}
}
