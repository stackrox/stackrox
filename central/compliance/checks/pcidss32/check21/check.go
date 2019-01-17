package check21

import (
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "PCI_DSS_3_2:2_1"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.DeploymentKind,
		nil,
		clusterIsCompliant)
}

// It's StackRox bro. Come on.
func clusterIsCompliant(ctx framework.ComplianceContext) {
	framework.Passf(ctx, "StackRox either randomly generates a strong admin password, or a user supplies one, for every deployment")
}
