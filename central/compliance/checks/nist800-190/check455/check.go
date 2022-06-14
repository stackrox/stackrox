package check455

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/policyfields"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const (
	standardID = "NIST_800_190:4_5_5"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              pkgFramework.DeploymentKind,
			DataDependencies:   []string{"Deployments", "Policies"},
			InterpretationText: interpretationText,
		},
		checkNIST455)
}

func checkNIST455(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	policyEnabled := false
	for _, policy := range policies {
		if !policyfields.ContainsVolumeSourceField(policy) {
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
