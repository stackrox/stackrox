package common

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stackrox/stackrox/pkg/docker/types"
)

// NonDockerRuntimeSkipList returns the evidence if the docker runtime is not Docker
func NonDockerRuntimeSkipList() []*storage.ComplianceResultValue_Evidence {
	return SkipList("Node does not use Docker container runtime")
}

// CheckWithDockerData returns a check that runs on each node with access to docker data.
func CheckWithDockerData(f func(data *types.Data) []*storage.ComplianceResultValue_Evidence) standards.Check {
	return func(data *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
		if data.DockerData == nil {
			return NonDockerRuntimeSkipList()
		}
		return f(data.DockerData)
	}
}
