package check412

import (
	"strings"

	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	standardID = "NIST_800_190:4_1_2"
)

var (
	log = logging.New("NIST_800_190:4_1_2")
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Policies", "ImageIntegrations", "ProcessIndicators", "Deployments"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			checkNIST412(ctx)
		})
}

func checkNIST412(ctx framework.ComplianceContext) {
	checkSSHPortAndProcesses(ctx)
	checkPrivilegedCategoryPolicies(ctx)
	common.IsImageScannerInUse(ctx)
	common.CheckBuildTimePolicyEnforced(ctx)
}

func checkSSHPortAndProcesses(ctx framework.ComplianceContext) {
	// Map process indicators to deployments.
	deploymentIDToIndicators := make(map[string][]*storage.ProcessIndicator)
	for _, indicator := range ctx.Data().ProcessIndicators() {
		indicators := deploymentIDToIndicators[indicator.GetDeploymentId()]
		indicators = append(indicators, indicator)
		deploymentIDToIndicators[indicator.GetDeploymentId()] = indicators
	}

	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		if deploymentHasSSHProcess(deploymentIDToIndicators, deployment) && !sshPolicyEnforced(ctx) {
			framework.Fail(ctx, "Deployment has ssh process running and no policy to enforce against them.")
		} else if deploymentHasSSHProcess(deploymentIDToIndicators, deployment) && sshPolicyEnforced(ctx) {
			log.Errorf("SSH Policy is being enforced but found ssh process running in deployment %s", deployment.GetId())
			framework.Fail(ctx, "Deployment has ssh process running.")
		} else {
			framework.Pass(ctx, "Deployment has no ssh process running.")
		}
	})
}

func checkPrivilegedCategoryPolicies(ctx framework.ComplianceContext) {
	common.CheckAnyPolicyInCategoryEnforced(ctx, "Privileges")
	common.CheckAnyPolicyInCategoryEnforced(ctx, "Vulnerability Management")
}

// deploymentHasSSHProcess returns true if the deployment has ssh process running.
func deploymentHasSSHProcess(deploymentToIndicators map[string][]*storage.ProcessIndicator, deployment *storage.Deployment) bool {
	for deploymentID, indicators := range deploymentToIndicators {
		if deploymentID != deployment.GetId() {
			continue
		}
		for _, indicator := range indicators {
			if strings.Contains(indicator.GetSignal().GetExecFilePath(), "ssh") ||
				strings.Contains(indicator.GetSignal().GetExecFilePath(), "sshd") {
				return true
			}
		}
	}
	return false
}

// sshPolicyEnforced checks if there is a policy to detect and enforce
// ssh daemons running in deployments.
func sshPolicyEnforced(ctx framework.ComplianceContext) bool {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		if !policyHasSSH(p) {
			continue
		}
		return common.IsPolicyEnforced(p)
	}
	return false
}

func policyHasSSH(policy *storage.Policy) bool {
	return policy.GetFields() != nil && policy.GetFields().GetProcessPolicy() != nil &&
		policy.GetFields().GetProcessPolicy().GetName() == "sshd"
}
