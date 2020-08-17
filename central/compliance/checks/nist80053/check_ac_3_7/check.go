package checkac37

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	pkgCommon "github.com/stackrox/rox/pkg/compliance/checks/common"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:AC_3_(7)"

	interpretationText = pkgCommon.IsRBACConfiguredCorrectlyInterpretation
)

func init() {
	framework.MustRegisterCheckIfFlagDisabled(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              pkgFramework.ClusterKind,
			DataDependencies:   []string{"Deployments", "HostScraped"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.IsRBACConfiguredCorrectly(ctx)
		})
}
