package checkcm7

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyfields"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
	"github.com/stackrox/rox/pkg/stringutils"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:CM_7"

	interpretationText = `This control requires that unnecessary features be prohibited or restricted.

For this control, StackRox validates that at least one policy is enabled and enforced based on each of:
  1) port exposure or service exposure level, and
  2) runtime behavior.`
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
			var portExposePolicy, runtimePolicy string
			for name, p := range policies {
				if !common.IsPolicyEnabled(p) || !common.IsPolicyEnforced(p) {
					continue
				}
				if portExposePolicy == "" && policyfields.ContainsPortOrPortExposureFields(p) {
					portExposePolicy = name
				}
				if runtimePolicy == "" && common.PolicyIsInLifecycleStage(p, storage.LifecycleStage_RUNTIME) {
					runtimePolicy = name
				}
			}
			if stringutils.AllNotEmpty(portExposePolicy, runtimePolicy) {
				framework.Passf(ctx, "At least one policy regarding port exposure/exposure level (%q) and at least one runtime policy (%q) are enabled and enforced", portExposePolicy, runtimePolicy)
				return
			}
			framework.Fail(ctx, "Required, but could not find, at least one policy regarding port exposure and at least one runtime policy that is enabled and enforced")
		})
}
