package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetProcessIndicator returns a Mock Process ProcessIndicator
func GetProcessIndicator() *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:           "b3523d84-ac1a-4daa-a908-62d196c5a741",
		DeploymentId: GetDeployment().GetId(),
		Signal: &storage.ProcessSignal{
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "bin",
		},
	}
}
