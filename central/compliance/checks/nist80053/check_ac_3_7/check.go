package checkac37

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/checks/kubernetes"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:AC_3_(7)"

	interpretationText = common.IsRBACConfiguredCorrectlyInterpretation
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.NodeKind,
			DataDependencies:   []string{"Deployments", "HostScraped"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			kubernetes.MasterAPIServerCommandLine("NIST_SP_800_53_Rev_4", "authorization-mode", "RBAC", "RBAC", common.Contains).Run(ctx)
		})
}
