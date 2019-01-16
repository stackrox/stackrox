package check412

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterNewCheck(
		"NIST-800-190:4.1.2",
		framework.ClusterKind,
		[]string{"Policies", "ImageIntegrations"},
		func(ctx framework.ComplianceContext) {
			checkNIST412(ctx)
		})
}

func checkNIST412(ctx framework.ComplianceContext) {
	checkSSHPolicies(ctx)
	checkPrivilegedCategoryPolicies(ctx)
	common.CheckImageScannerInUse(ctx)
	common.CheckBuildTimePolicyEnforced(ctx)
}

func checkSSHPolicies(ctx framework.ComplianceContext) {
	common.CheckPolicyInUse(ctx, "Secure Shell (ssh) Port Exposed")
	common.CheckPolicyInUse(ctx, "Secure Shell Server (sshd) Execution")
}

func checkPrivilegedCategoryPolicies(ctx framework.ComplianceContext) {
	common.CheckAnyPolicyInCategoryEnforced(ctx, "Privileges")
	common.CheckAnyPolicyInCategoryEnforced(ctx, "Vulnerability Management")
}
