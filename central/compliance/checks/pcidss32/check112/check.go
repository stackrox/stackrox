package check112

import (
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "PCI_DSS_3_2:1_1_2"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.ClusterKind,
		nil,
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	framework.Pass(ctx, passText())
}
