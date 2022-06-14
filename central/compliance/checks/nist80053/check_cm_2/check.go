package checkcm2

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/storage"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:CM_2"

	phase = storage.LifecycleStage_DEPLOY
)

var (
	interpretationText = `This control requires baseline configurations to be defined and validated.

For this control, ` + common.AnyPolicyInLifeCycleInterpretation(phase)
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
			common.CheckAnyPolicyInLifeCycle(ctx, phase)
		})
}
