package kubernetes

import (
	"github.com/stackrox/stackrox/pkg/compliance/checks/common"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("5_3_1"): common.NoteCheck("Ensure that the CNI in use supports Network Policies"),
		// TODO: @boo - implement the check below
		standards.CISKubeCheckName("5_3_2"): common.NoteCheck("Ensure that all Namespaces have Network Policies defined"),
	})
}
