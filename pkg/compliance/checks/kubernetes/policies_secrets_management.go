package kubernetes

import (
	"github.com/stackrox/rox/pkg/compliance/checks/common"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

func init() {
	standards.RegisterChecksForStandard(standards.CISKubernetes, map[string]*standards.CheckAndMetadata{
		standards.CISKubeCheckName("5_4_1"): common.NoteCheck("Prefer using secrets as files over secrets as environment variables"),
		standards.CISKubeCheckName("5_4_2"): common.NoteCheck("Consider external secret storage"),
	})
}
