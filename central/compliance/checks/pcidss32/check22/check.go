package check22

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "PCI_DSS_3_2:2_2"

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              framework.ClusterKind,
			DataDependencies:   []string{"CISBenchmarks"},
			InterpretationText: interpretationText,
		},
		common.CISBenchmarksSatisfied)
}
