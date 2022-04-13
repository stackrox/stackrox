package checkir61

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/storage"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:IR_6_(1)"

	interpretationText = `This control requires the use of automated mechanisms to report information security incidents.

For this control, StackRox checks that at least one runtime policy is set to notify at least one workflow tool.`
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Policies"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			policies := ctx.Data().Policies()
			for name, p := range policies {
				if !common.IsPolicyEnabled(p) {
					continue
				}
				if !common.PolicyIsInLifecycleStage(p, storage.LifecycleStage_RUNTIME) {
					continue
				}
				if len(p.GetNotifiers()) == 0 {
					continue
				}
				framework.Passf(ctx, "Policy %q is a runtime policy, set to send notifications", name)
				return
			}
			framework.Fail(ctx, "No runtime policies were set to notify a workflow tool")
		})
}
