package check412

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyfields"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sliceutils"
)

const (
	standardID = "NIST_800_190:4_1_2"
)

var (
	log = logging.ModuleForName("NIST_800_190:4_1_2").Logger()
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              pkgFramework.ClusterKind,
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
	common.CheckImageScannerInUseByCluster(ctx)
	common.CheckAnyPolicyInLifecycleStageEnforced(ctx, storage.LifecycleStage_BUILD)
}

func checkSSHPortAndProcesses(ctx framework.ComplianceContext) {
	// Map process indicators to deployments.
	deploymentIDToIndicators := make(map[string][]*storage.ProcessIndicator)
	for _, indicator := range ctx.Data().SSHProcessIndicators() {
		deploymentIDToIndicators[indicator.GetDeploymentId()] = append(deploymentIDToIndicators[indicator.GetDeploymentId()], indicator)
	}

	framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
		sshProcessRunning := deploymentHasSSHProcess(deploymentIDToIndicators, deployment)
		sshEnforced := sshPolicyEnforced(ctx)
		if sshProcessRunning && !sshEnforced {
			framework.Fail(ctx, "Deployment has ssh process running and no policy to enforce against them.")
		} else if sshProcessRunning && sshEnforced {
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
	indicators := deploymentToIndicators[deployment.GetId()]
	return len(indicators) > 0
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
	return sliceutils.Find(policyfields.GetProcessNames(policy), "sshd") >= 0
}
