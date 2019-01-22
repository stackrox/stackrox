package check444

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

const (
	standardID = "NIST_800_190:4_4_4"
)

func init() {
	framework.MustRegisterNewCheck(
		standardID,
		framework.DeploymentKind,
		[]string{"Alerts", "Deployments"},
		checkNIST444)
}

func checkNIST444(ctx framework.ComplianceContext) {
	common.AlertsForDeployments(ctx, storage.LifecycleStage_RUNTIME)
}
