package check421

import (
	"github.com/stackrox/rox/central/compliance/checks/common"
	"github.com/stackrox/rox/central/compliance/framework"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

const (
	checkID = "NIST_800_190:4_2_1"
)

func init() {
	framework.MustRegisterCheckIfFlagDisabled(
		framework.CheckMetadata{
			ID:                 checkID,
			Scope:              pkgFramework.NodeKind,
			DataDependencies:   []string{"HostScraped"},
			InterpretationText: interpretationText,
		},
		common.CheckNoInsecureRegistries)
}
