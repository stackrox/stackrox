package check23

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const checkID = "PCI_DSS_3_2:2_3"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              framework.ClusterKind,
			InterpretationText: interpretationText,
		},
		clusterIsCompliant)
}

func clusterIsCompliant(ctx framework.ComplianceContext) {
	framework.Pass(ctx, passText())
}
