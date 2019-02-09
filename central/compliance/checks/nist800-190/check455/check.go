package check455

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

const (
	standardID = "NIST_800_190:4_5_5"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              framework.DeploymentKind,
			DataDependencies:   []string{"Deployments", "Policies"},
			InterpretationText: interpretationText,
		},
		checkNIST455)
}

func checkNIST455(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	policyEnabled := false
	for _, policy := range policies {
		if policy.GetFields().GetVolumePolicy().GetSource() == "" {
			continue
		}

		if common.IsPolicyEnabled(policy) {
			common.CheckViolationsForPolicyByDeployment(ctx, policy)
			policyEnabled = true // set only once
		}
	}

	// None of the volume based policies are enabled
	if !policyEnabled {
		framework.ForEachDeployment(ctx, func(ctx framework.ComplianceContext, deployment *storage.Deployment) {
			framework.Fail(ctx, "No policies to check for sensitive host mounts")
		})
	}
}
