package kubernetes

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("5_5_1"): common.NoteCheck("Configure Image Provenance using ImagePolicyWebhook admission controller"),
	})
}
