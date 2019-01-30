package check1121

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

const checkID = "PCI_DSS_3_2:11_2_1"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.ClusterKind,
		[]string{"ImageIntegrations"},
		common.IsImageScannerInUse)
}
