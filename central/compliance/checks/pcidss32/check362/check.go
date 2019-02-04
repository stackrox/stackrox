package check362

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const checkID = "PCI_DSS_3_2:3_6_2"

func init() {
	framework.MustRegisterNewCheck(
		checkID,
		framework.ClusterKind,
		[]string{"Policies"},
		common.CheckSecretsInEnv)
}
