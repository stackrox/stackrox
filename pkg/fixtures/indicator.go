package fixtures

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
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
func GetScopedProcessIndicator(t *testing.T, ID string, clusterID string, namespace string) *storage.ProcessIndicator {
	indicator := &storage.ProcessIndicator{}
	require.NoError(t, testutils.FullInit(indicator, testutils.SimpleInitializer(), testutils.JSONFieldsFilter))
	indicator.Id = ID
	indicator.ClusterId = clusterID
	indicator.Namespace = namespace

	return indicator
}
