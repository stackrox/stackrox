package checkac24

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/features"
)

const (
	controlID = "NIST_SP_800_53:AC_24"

	interpretationText = `StackRox has visibility into the authentication configuration used in your Kubernetes
cluster. This data can indicate whether the cluster has been properly set up to enforce access control.`
)

func init() {
	framework.MustRegisterNewCheckIfFlagEnabled(
		framework.CheckMetadata{
			ID:                 controlID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"Deployments"},
			InterpretationText: interpretationText,
		},
		func(ctx framework.ComplianceContext) {
			common.IsRBACConfiguredCorrectly(ctx)
		}, features.NistSP800_53)
}
