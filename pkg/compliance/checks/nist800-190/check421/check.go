package check421

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/compliance/framework"
)

const (
	checkID = "4_2_1"
)

func init() {
	standards.RegisterChecksForStandard(standards.NIST800190, map[string]*standards.CheckAndMetadata{
		standards.NIST800190CheckName(checkID): {
			CheckFunc: common.CheckNoInsecureRegistries,
			Metadata: &standards.Metadata{
				InterpretationText: interpretationText,
				TargetKind:         framework.NodeKind,
			},
		},
	})
}
