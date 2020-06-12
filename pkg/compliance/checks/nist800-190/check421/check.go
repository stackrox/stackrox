package check421

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

const (
	checkID = "4_2_1"
)

func init() {
	standards.RegisterChecksForStandard(standards.NIST, map[string]*standards.CheckAndInterpretation{
		standards.NISTCheckName(checkID): {
			CheckFunc:          common.CheckNoInsecureRegistries,
			InterpretationText: interpretationText,
		},
	})
}
