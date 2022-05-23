package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
)

// GetProcessIndicator returns a mock ProcessIndicator.
func GetProcessIndicator() *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:           "b3523d84-ac1a-4daa-a908-62d196c5a741",
		DeploymentId: GetDeployment().GetId(),
		Signal: &storage.ProcessSignal{
			ContainerId:  "containerid",
			Name:         "apt-get",
			Args:         "install nmap",
			ExecFilePath: "bin",
			LineageInfo: []*storage.ProcessSignal_LineageInfo{
				{
					ParentUid:          22,
					ParentExecFilePath: "/bin/bash",
				},
				{
					ParentUid:          28,
					ParentExecFilePath: "/bin/curl",
				},
			},
		},
	}
}

// GetScopedProcessIndicator returns a mock ProcessIndicator belonging to the input scope.
func GetScopedProcessIndicator(ID string, clusterID string, namespace string) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:        ID,
		ClusterId: clusterID,
		Namespace: namespace,
	}
}
