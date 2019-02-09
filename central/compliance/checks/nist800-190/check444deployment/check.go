package check444deployment

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	standardID = "NIST_800_190:4_4_4_deployment"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 standardID,
			Scope:              framework.DeploymentKind,
			DataDependencies:   []string{"Deployments"},
			InterpretationText: interpretationText,
		},
		common.CheckDeploymentHasReadOnlyFSByDeployment)
}
