package checkcm5

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.NIST80053, map[string]*standards.CheckAndMetadata{
		// This is a partial check.  The evidence from this check will be folded together with evidence generated in central
		standards.NIST80053CheckName("CM_5"): common.MasterAPIServerCommandLine("authorization-mode", "RBAC", "RBAC", common.Contains),
	})
}
