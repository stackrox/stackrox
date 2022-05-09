package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
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
		Id:           ID,
		ClusterId:    clusterID,
		Namespace:    namespace,
		DeploymentId: ID,
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

// GetSACTestProcessIndicatorSet returns a set of mock ProcessIndicators that can be used
// for scoped access control sets.
// It will include:
// 9 Process indicators scoped to Cluster1, 3 to each Namespace A / B / C.
// 9 Process indicators scoped to Cluster2, 3 to each Namespace A / B / C.
// 9 Process indicators scoped to Cluster2, 3 to each Namespace A / B / C.
func GetSACTestProcessIndicatorSet() []*storage.ProcessIndicator {
	return []*storage.ProcessIndicator{
		scopedProcessIndicator(testconsts.Cluster1, testconsts.NamespaceA),
		scopedProcessIndicator(testconsts.Cluster1, testconsts.NamespaceA),
		scopedProcessIndicator(testconsts.Cluster1, testconsts.NamespaceA),
		scopedProcessIndicator(testconsts.Cluster1, testconsts.NamespaceB),
		scopedProcessIndicator(testconsts.Cluster1, testconsts.NamespaceB),
		scopedProcessIndicator(testconsts.Cluster1, testconsts.NamespaceB),
		scopedProcessIndicator(testconsts.Cluster1, testconsts.NamespaceC),
		scopedProcessIndicator(testconsts.Cluster1, testconsts.NamespaceC),
		scopedProcessIndicator(testconsts.Cluster1, testconsts.NamespaceC),
		scopedProcessIndicator(testconsts.Cluster2, testconsts.NamespaceA),
		scopedProcessIndicator(testconsts.Cluster2, testconsts.NamespaceA),
		scopedProcessIndicator(testconsts.Cluster2, testconsts.NamespaceA),
		scopedProcessIndicator(testconsts.Cluster2, testconsts.NamespaceB),
		scopedProcessIndicator(testconsts.Cluster2, testconsts.NamespaceB),
		scopedProcessIndicator(testconsts.Cluster2, testconsts.NamespaceB),
		scopedProcessIndicator(testconsts.Cluster2, testconsts.NamespaceC),
		scopedProcessIndicator(testconsts.Cluster2, testconsts.NamespaceC),
		scopedProcessIndicator(testconsts.Cluster2, testconsts.NamespaceC),
		scopedProcessIndicator(testconsts.Cluster3, testconsts.NamespaceA),
		scopedProcessIndicator(testconsts.Cluster3, testconsts.NamespaceA),
		scopedProcessIndicator(testconsts.Cluster3, testconsts.NamespaceA),
		scopedProcessIndicator(testconsts.Cluster3, testconsts.NamespaceB),
		scopedProcessIndicator(testconsts.Cluster3, testconsts.NamespaceB),
		scopedProcessIndicator(testconsts.Cluster3, testconsts.NamespaceB),
		scopedProcessIndicator(testconsts.Cluster3, testconsts.NamespaceC),
		scopedProcessIndicator(testconsts.Cluster3, testconsts.NamespaceC),
		scopedProcessIndicator(testconsts.Cluster3, testconsts.NamespaceC),
	}
}

func scopedProcessIndicator(clusterID, namespace string) *storage.ProcessIndicator {
	return GetScopedProcessIndicator(uuid.NewV4().String(), clusterID, namespace)
}
