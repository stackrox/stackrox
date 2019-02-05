package check455

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	standardID = "NIST_800_190:4_5_5"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Policies"},
			InterpretationText: interpretationText,
		},
		checkNIST455)
}

func checkNIST455(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	policyFound := false
	policyEnabled := false
	for _, policy := range policies {
		if policy.GetFields().GetVolumePolicy().GetSource() != "" {
			policyFound = true
			if common.IsPolicyEnabled(policy) {
				policyEnabled = true
			}
		}
	}
	if !policyFound {
		framework.Fail(ctx, "No policies to detect sensitive host mounts was found")
	} else if policyFound && !policyEnabled {
		framework.Fail(ctx, "Policy to detect sensitive host mounts is not enabled")
	} else {
		framework.Pass(ctx, "Policies to detect sensitive host mounts is enabled")
	}
}
