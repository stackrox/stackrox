package checkac14

import (
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.NIST80053, map[string]*standards.CheckAndMetadata{
		// This is a partial check.  The evidence from this check will be folded together with evidence generated in central
		standards.NIST80053CheckName("AC_14"): common.MasterAPIServerCommandLine("authorization-mode", "RBAC", "RBAC", common.Contains),
	})
}
