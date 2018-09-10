package fixtures

import "github.com/stackrox/rox/generated/api/v1"

// GetProcessIndicator returns a Mock Process ProcessIndicator
func GetProcessIndicator() *v1.ProcessIndicator {
	return &v1.ProcessIndicator{
		Id:           "b3523d84-ac1a-4daa-a908-62d196c5a741",
		DeploymentId: GetDeployment().GetId(),
		Signal: &v1.ProcessSignal{
			Name:         "apt-get",
			CommandLine:  "install nmap",
			ExecFilePath: "bin",
		},
	}
}
