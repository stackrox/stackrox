package checksi38

import (
	"github.com/stackrox/stackrox/central/compliance/checks/common"
	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/storage"
	pkgFramework "github.com/stackrox/stackrox/pkg/compliance/framework"
)

const (
	controlID = "NIST_SP_800_53_Rev_4:SI_3_(8)"

	phase = storage.LifecycleStage_RUNTIME
)

var (
	interpretationText = `This control requires that unauthorized operating system commands be detected and reported or blocked.

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
			framework.Pass(ctx, "The StackRox Kubernetes Security Platform is installed and tracking potential unauthorized commands.")
			common.CheckAnyPolicyInLifeCycle(ctx, phase)
		})
}
