package check24

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

const checkID = "PCI_DSS_3_2:2_4"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.DeploymentKind,
		nil,
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	framework.Pass(ctx, passText())
}
