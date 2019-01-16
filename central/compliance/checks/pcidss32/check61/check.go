package check61

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

func init() {
	framework.MustRegisterNewCheck(
		"PCI_DSS_3_2:6_1",
		framework.ClusterKind,
		nil,
		common.CheckImageScannerInUse)
}
