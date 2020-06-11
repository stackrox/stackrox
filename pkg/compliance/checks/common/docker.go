package common

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
	"github.com/stackrox/rox/pkg/docker/types"
)

// CheckWithDockerData returns a check that runs on each node with access to docker data.
func CheckWithDockerData(f func(data *types.Data) []*storage.ComplianceResultValue_Evidence) standards.Check {
	return func(data *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
		// TODO: Figure out how to abort the compliance run
		if data.DockerData == nil {
			return nil
		}
		return f(data.DockerData)
	}
}
