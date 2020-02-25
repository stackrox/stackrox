package checkir61

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
)

const (
	controlID = "NIST_SP_800_53:IR_6_(1)"

	interpretationText = `TODO`
)

func init() {
	framework.MustRegisterNewCheckIfFlagEnabled(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.ClusterKind,
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
			framework.Fail(ctx, "Required at least one runtime policy that is set to notify at least one workflow tool")
		}, features.NistSP800_53)
}
