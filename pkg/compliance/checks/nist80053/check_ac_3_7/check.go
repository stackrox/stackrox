package checkac37

import (
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.NIST80053, map[string]*standards.CheckAndMetadata{
		standards.NIST80053CheckName("AC_3_(7)"): common.MasterAPIServerRBACConfigurationCommandLine(),
	})
}
