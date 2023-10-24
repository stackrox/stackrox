package fixtures

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
)

// GetPod returns a mock Pod
func GetPod() *storage.Pod {
	return &storage.Pod{
		Id:           "nginx-7db9fccd9b-92hfs",
		DeploymentId: GetDeployment().GetId(),
		ClusterId:    "prod cluster",
		Namespace:    "stackrox",
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

// GetPod1 returns a mock Pod
func GetPod1() *storage.Pod {
	return &storage.Pod{
		Id:           fixtureconsts.PodUID1,
		Name:         fixtureconsts.PodName1,
		DeploymentId: GetDeployment().GetId(),
		ClusterId:    "prod cluster",
		Namespace:    "stackrox",
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

// GetScopedPod returns a mock Pod belonging to the input scope.
func GetScopedPod(ID string, clusterID string, namespace string) *storage.Pod {
	return &storage.Pod{
		Id:        ID,
		ClusterId: clusterID,
		Namespace: namespace,
	}
}
