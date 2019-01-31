package check310d

import (
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "HIPAA_164:310_d"

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
