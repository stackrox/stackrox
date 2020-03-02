package checksc6

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	controlID = "NIST_SP_800_53:SC_6"

	interpretationText = `This control requires resource management practices to protect availability.

For this control, StackRox checks that at least one policy requiring CPU limits and memory limits is enabled and enforced.`
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
			var cpuLimitPolicy, memLimitPolicy string
			for name, p := range policies {
				if !(common.IsPolicyEnforced(p) && common.IsPolicyEnabled(p)) {
					continue
				}
				if p.GetFields().GetContainerResourcePolicy().GetCpuResourceLimit() != nil {
					cpuLimitPolicy = name
				}
				if p.GetFields().GetContainerResourcePolicy().GetMemoryResourceLimit() != nil {
					memLimitPolicy = name
				}
			}
			if stringutils.AllNotEmpty(cpuLimitPolicy, memLimitPolicy) {
				framework.Passf(ctx, "There is at least one policy implemented and enforced for CPU resource limit (%q) and memory resource limit (%q)", cpuLimitPolicy, memLimitPolicy)
				return
			}
			framework.Fail(ctx, "At least one policy must be implemented and enforced for CPU resource limits and memory resource limits")
		}, features.NistSP800_53)
}
