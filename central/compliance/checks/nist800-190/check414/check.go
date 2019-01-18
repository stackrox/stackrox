package check414

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

const (
	standardID = "NIST_800_190:4_1_4"
)

func init() {
	framework.MustRegisterNewCheck(
		standardID,
		framework.ClusterKind,
		[]string{"Policies"},
		func(ctx framework.ComplianceContext) {
			checkNIST414(ctx)
		})
}

func checkNIST414(ctx framework.ComplianceContext) {
	checkEnvSecretPolicyEnforced(ctx)
}

func checkEnvSecretPolicyEnforced(ctx framework.ComplianceContext) {
	policies := ctx.Data().Policies()
	for _, p := range policies {
		a := common.NewAnder(common.IsPolicyEnabled(p), common.IsPolicyEnforced(p), doesPolicyHaveEnv(p), doesPolicyHaveSecretKeyValue(p))

		if a.Execute() {
			framework.Passf(ctx, "Policy '%s' enabled and enforced", p.GetName())
			return
		}
	}

	framework.Fail(ctx, "Policy that disallows the use of secrets in environment variables not found")
}

func doesPolicyHaveEnv(p *storage.Policy) common.Andable {
	return func() bool {
		return p.GetFields() != nil && p.GetFields().GetEnv() != nil
	}
}

func doesPolicyHaveSecretKeyValue(p *storage.Policy) common.Andable {
	return func() bool {
		return p.GetFields().GetEnv().GetKey() != ".*SECRET.*" && p.GetFields().GetEnv().GetValue() != ""
	}
}
