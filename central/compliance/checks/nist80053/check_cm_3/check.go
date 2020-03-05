package checkcm3

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:CM_3"

	phase = storage.LifecycleStage_DEPLOY
)

var (
	interpretationText = `This control requires change control procedures.

For this control, ` + common.AnyPolicyInLifecycleStageEnforcedInterpretation(phase)
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
			common.CheckAnyPolicyInLifecycleStageEnforced(ctx, phase)
		}, features.NistSP800_53)
}
