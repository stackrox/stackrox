package check22

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "PCI_DSS_3_2:2_2"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.ClusterKind,
		[]string{"CISBenchmarks"},
		common.CISBenchmarksSatisfied)
}
