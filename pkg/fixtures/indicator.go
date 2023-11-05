package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/process/id"
)

// GetProcessIndicator returns a mock ProcessIndicator.
func GetProcessIndicator() *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:           "b3523d84-ac1a-4daa-a908-62d196c5a741",
		DeploymentId: GetDeployment().GetId(),
		Namespace:    fixtureconsts.Namespace1,
		ClusterId:    fixtureconsts.Cluster1,
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

// GetProcessIndicator1 returns a mock ProcessIndicator.
func GetProcessIndicator1() *storage.ProcessIndicator {
	pi := &storage.ProcessIndicator{
		Id:            "b3523d84-ac1a-4daa-a908-62d196c5a741",
		DeploymentId:  fixtureconsts.Deployment1,
		ContainerName: "containername",
		Namespace:     fixtureconsts.Namespace1,
		ClusterId:     fixtureconsts.Cluster1,
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
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
	id.SetIndicatorID(pi)

	return pi
}

// GetProcessIndicator2 returns a mock ProcessIndicator.
func GetProcessIndicator2() *storage.ProcessIndicator {
	pi := &storage.ProcessIndicator{
		Id:            "b3523d84-ac1a-4daa-a908-62d196c5a741",
		DeploymentId:  fixtureconsts.Deployment1,
		ContainerName: "containername",
		Namespace:     fixtureconsts.Namespace1,
		ClusterId:     fixtureconsts.Cluster1,
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		Signal: &storage.ProcessSignal{
			ContainerId:  "containerid",
			Name:         "dnf",
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
	id.SetIndicatorID(pi)

	return pi
}

// GetProcessIndicator3 returns a mock ProcessIndicator.
func GetProcessIndicator3() *storage.ProcessIndicator {
	pi := &storage.ProcessIndicator{
		Id:            "b3523d84-ac1a-4daa-a908-62d196c5a741",
		DeploymentId:  fixtureconsts.Deployment1,
		ContainerName: "containername",
		Namespace:     fixtureconsts.Namespace1,
		ClusterId:     fixtureconsts.Cluster1,
		PodId:         fixtureconsts.PodName2,
		PodUid:        fixtureconsts.PodUID2,
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
	id.SetIndicatorID(pi)

	return pi
}

// GetProcessIndicator4 returns a mock ProcessIndicator.
func GetProcessIndicator4() *storage.ProcessIndicator {
	pi := &storage.ProcessIndicator{
		Id:            "b3523d84-ac1a-4daa-a908-62d196c5a741",
		DeploymentId:  fixtureconsts.Deployment6,
		ContainerName: "containername",
		Namespace:     fixtureconsts.Namespace1,
		ClusterId:     fixtureconsts.Cluster1,
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
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
	id.SetIndicatorID(pi)

	return pi
}

// GetProcessIndicator5 returns a mock ProcessIndicator.
func GetProcessIndicator5() *storage.ProcessIndicator {
	pi := &storage.ProcessIndicator{
		Id:            "b3523d84-ac1a-4daa-a908-62d196c5a741",
		DeploymentId:  fixtureconsts.Deployment5,
		ContainerName: "containername",
		Namespace:     fixtureconsts.Namespace1,
		ClusterId:     fixtureconsts.Cluster1,
		PodId:         fixtureconsts.PodName1,
		PodUid:        fixtureconsts.PodUID1,
		Signal: &storage.ProcessSignal{
			ContainerId:  "containerid",
			Name:         "dnf",
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
	id.SetIndicatorID(pi)

	return pi
}

// GetProcessIndicator6 returns a mock ProcessIndicator.
func GetProcessIndicator6() *storage.ProcessIndicator {
	pi := &storage.ProcessIndicator{
		Id:            "b3523d84-ac1a-4daa-a908-62d196c5a741",
		DeploymentId:  fixtureconsts.Deployment3,
		ContainerName: "containername",
		Namespace:     fixtureconsts.Namespace1,
		ClusterId:     fixtureconsts.Cluster1,
		PodId:         fixtureconsts.PodName2,
		PodUid:        fixtureconsts.PodUID2,
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
	id.SetIndicatorID(pi)

	return pi
}

// GetScopedProcessIndicator returns a mock ProcessIndicator belonging to the input scope.
func GetScopedProcessIndicator(ID string, clusterID string, namespace string) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Id:        ID,
		ClusterId: clusterID,
		Namespace: namespace,
	}
}
