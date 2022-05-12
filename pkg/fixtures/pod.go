package fixtures

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

// GetPod returns a mock Pod
func GetPod() *storage.Pod {
	return GetScopedPod("nginx-7db9fccd9b-92hfs", GetDeployment().GetId(),
		"prod cluster", "stackrox")
}

// GetScopedPod returns a mock Pod belonging to the input scope.
func GetScopedPod(ID string, deploymentID string, clusterID string, namespace string) *storage.Pod {
	return &storage.Pod{
		Id:           ID,
		DeploymentId: deploymentID,
		ClusterId:    clusterID,
		Namespace:    namespace,
		Started: &types.Timestamp{
			Seconds: 0,
		},
		LiveInstances: []*storage.ContainerInstance{
			{
				InstanceId: &storage.ContainerInstanceID{
					Id: "containerid",
				},
				ContainerName: "containername",
				ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started: &types.Timestamp{
					Seconds: 2,
				},
			},
			{
				InstanceId: &storage.ContainerInstanceID{
					Id: "othercontainerid",
				},
				ContainerName: "othercontainername",
				ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started: &types.Timestamp{
					Seconds: 3,
				},
			},
		},
		TerminatedInstances: []*storage.Pod_ContainerInstanceList{
			{
				Instances: []*storage.ContainerInstance{
					{
						InstanceId: &storage.ContainerInstanceID{
							Id: "containeridfirst",
						},
						ContainerName: "containername",
						ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started: &types.Timestamp{
							Seconds: 0,
						},
						Finished: &types.Timestamp{
							Seconds: 1,
						},
					},
				},
			},
			{
				Instances: []*storage.ContainerInstance{
					{
						InstanceId: &storage.ContainerInstanceID{
							Id: "othercontainerid",
						},
						ContainerName: "othercontainername",
						ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started: &types.Timestamp{
							Seconds: 1,
						},
						Finished: &types.Timestamp{
							Seconds: 2,
						},
					},
				},
			},
			{
				Instances: []*storage.ContainerInstance{
					{
						InstanceId: &storage.ContainerInstanceID{
							Id: "containeridinitfirst",
						},
						ContainerName: "containerinitname",
						ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started: &types.Timestamp{
							Seconds: 0,
						},
						Finished: &types.Timestamp{
							Seconds: 1,
						},
						ExitCode:          137,
						TerminationReason: "Error",
					},
					{
						InstanceId: &storage.ContainerInstanceID{
							Id: "containeridinitsecond",
						},
						ContainerName: "containerinitname",
						ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started: &types.Timestamp{
							Seconds: 1,
							Nanos:   200,
						},
						Finished: &types.Timestamp{
							Seconds: 1,
							Nanos:   800,
						},
						ExitCode:          0,
						TerminationReason: "Completed",
					},
				},
			},
		},
	}
}

// GetSACTestPodSet returns a set of mock Pods that can be used
// for scoped access control sets.
// It will include:
// 9 Process indicators scoped to Cluster1, 3 to each Namespace A / B / C.
// 9 Process indicators scoped to Cluster2, 3 to each Namespace A / B / C.
// 9 Process indicators scoped to Cluster3, 3 to each Namespace A / B / C.
func GetSACTestPodSet() []*storage.Pod {
	return []*storage.Pod{
		scopedPod(testconsts.Cluster1, testconsts.NamespaceA),
		scopedPod(testconsts.Cluster1, testconsts.NamespaceA),
		scopedPod(testconsts.Cluster1, testconsts.NamespaceA),
		scopedPod(testconsts.Cluster1, testconsts.NamespaceB),
		scopedPod(testconsts.Cluster1, testconsts.NamespaceB),
		scopedPod(testconsts.Cluster1, testconsts.NamespaceB),
		scopedPod(testconsts.Cluster1, testconsts.NamespaceC),
		scopedPod(testconsts.Cluster1, testconsts.NamespaceC),
		scopedPod(testconsts.Cluster1, testconsts.NamespaceC),
		scopedPod(testconsts.Cluster2, testconsts.NamespaceA),
		scopedPod(testconsts.Cluster2, testconsts.NamespaceA),
		scopedPod(testconsts.Cluster2, testconsts.NamespaceA),
		scopedPod(testconsts.Cluster2, testconsts.NamespaceB),
		scopedPod(testconsts.Cluster2, testconsts.NamespaceB),
		scopedPod(testconsts.Cluster2, testconsts.NamespaceB),
		scopedPod(testconsts.Cluster2, testconsts.NamespaceC),
		scopedPod(testconsts.Cluster2, testconsts.NamespaceC),
		scopedPod(testconsts.Cluster2, testconsts.NamespaceC),
		scopedPod(testconsts.Cluster3, testconsts.NamespaceA),
		scopedPod(testconsts.Cluster3, testconsts.NamespaceA),
		scopedPod(testconsts.Cluster3, testconsts.NamespaceA),
		scopedPod(testconsts.Cluster3, testconsts.NamespaceB),
		scopedPod(testconsts.Cluster3, testconsts.NamespaceB),
		scopedPod(testconsts.Cluster3, testconsts.NamespaceB),
		scopedPod(testconsts.Cluster3, testconsts.NamespaceC),
		scopedPod(testconsts.Cluster3, testconsts.NamespaceC),
		scopedPod(testconsts.Cluster3, testconsts.NamespaceC),
	}
}

func scopedPod(clusterID, namespace string) *storage.Pod {
	return GetScopedPod(uuid.NewV4().String(), uuid.NewV4().String(), clusterID, namespace)
}
