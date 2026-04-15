package checkac24

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func RegisterCheckAC24() {
	standards.RegisterChecksForStandard(standards.NIST80053, map[string]*standards.CheckAndMetadata{
		standards.NIST80053CheckName("AC_24"): common.MasterAPIServerRBACConfigurationCommandLine(),
	})
}
