package check308a3iia

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

const checkID = "HIPAA_164:308_a_3_ii_a"

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
