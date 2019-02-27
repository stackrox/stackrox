package check421

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
)

const (
	checkID = "NIST_800_190:4_2_1"
)

func init() {
	framework.MustRegisterNewCheck(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              framework.NodeKind,
			DataDependencies:   []string{"HostScraped"},
			InterpretationText: interpretationText,
		},
		common.CheckNoInsecureRegistries)
}
