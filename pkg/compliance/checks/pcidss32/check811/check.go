package check811

import (
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.PCIDSS32, map[string]*standards.CheckAndMetadata{
		standards.PCIDSS32CheckName("8_1_1"): clusterIsCompliant(),
	})
}

func clusterIsCompliant() *standards.CheckAndMetadata {
	// This is a partial check.  The evidence from this check will be folded together with evidence generated in central
	checkAndMetadata := common.MasterAPIServerRBACConfigurationCommandLine()
	checkAndMetadata.Metadata.InterpretationText = interpretationText
	return checkAndMetadata
}
