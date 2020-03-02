package checksi38

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
)

const (
	controlID = "NIST_SP_800_53:SI_3_(8)"

	phase = storage.LifecycleStage_RUNTIME
)

var (
	interpretationText = `This control requires that unauthorized operating system commands be detected and reported or blocked.

For this control, ` + common.AnyPolicyInLifeCycleInterpretation(phase)
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
			framework.Pass(ctx, "The StackRox Kubernetes Security Platform is installed and tracking potential unauthorized commands.")
			common.CheckAnyPolicyInLifeCycle(ctx, phase)
		}, features.NistSP800_53)
}
