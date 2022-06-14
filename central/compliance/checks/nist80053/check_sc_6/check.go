package checksc6

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/pkg/booleanpolicy/policyfields"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
	"github.com/stackrox/stackrox/pkg/stringutils"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:SC_6"

	interpretationText = `This control requires resource management practices to protect availability.

For this control, StackRox checks that at least one policy requiring CPU limits and memory limits is enabled and enforced.`
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
			var cpuLimitPolicy, memLimitPolicy string
			for name, p := range policies {
				if !(common.IsPolicyEnforced(p) && common.IsPolicyEnabled(p)) {
					continue
				}
				if policyfields.ContainsCPUResourceLimit(p) {
					cpuLimitPolicy = name
				}
				if policyfields.ContainsMemResourceLimit(p) {
					memLimitPolicy = name
				}
			}
			if stringutils.AllNotEmpty(cpuLimitPolicy, memLimitPolicy) {
				framework.Passf(ctx, "There is at least one policy implemented and enforced for CPU resource limit (%q) and memory resource limit (%q)", cpuLimitPolicy, memLimitPolicy)
				return
			}
			framework.Fail(ctx, "At least one policy must be implemented and enforced for CPU resource limits and memory resource limits")
		})
}
