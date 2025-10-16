package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protocompat"
)

// GetPod returns a mock Pod
func GetPod() *storage.Pod {
	return storage.Pod_builder{
		Id:           "nginx-7db9fccd9b-92hfs",
		DeploymentId: GetDeployment().GetId(),
		ClusterId:    "prod cluster",
		Namespace:    "stackrox",
		Started:      protocompat.GetProtoTimestampFromSeconds(0),
		LiveInstances: []*storage.ContainerInstance{
			storage.ContainerInstance_builder{
				InstanceId: storage.ContainerInstanceID_builder{
					Id: "containerid",
				}.Build(),
				ContainerName: "containername",
				ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started:       protocompat.GetProtoTimestampFromSeconds(2),
			}.Build(),
			storage.ContainerInstance_builder{
				InstanceId: storage.ContainerInstanceID_builder{
					Id: "othercontainerid",
				}.Build(),
				ContainerName: "othercontainername",
				ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started:       protocompat.GetProtoTimestampFromSeconds(3),
			}.Build(),
		},
		TerminatedInstances: []*storage.Pod_ContainerInstanceList{
			storage.Pod_ContainerInstanceList_builder{
				Instances: []*storage.ContainerInstance{
					storage.ContainerInstance_builder{
						InstanceId: storage.ContainerInstanceID_builder{
							Id: "containeridfirst",
						}.Build(),
						ContainerName: "containername",
						ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started:       protocompat.GetProtoTimestampFromSeconds(0),
						Finished:      protocompat.GetProtoTimestampFromSeconds(1),
					}.Build(),
				},
			}.Build(),
			storage.Pod_ContainerInstanceList_builder{
				Instances: []*storage.ContainerInstance{
					storage.ContainerInstance_builder{
						InstanceId: storage.ContainerInstanceID_builder{
							Id: "othercontainerid",
						}.Build(),
						ContainerName: "othercontainername",
						ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started:       protocompat.GetProtoTimestampFromSeconds(1),
						Finished:      protocompat.GetProtoTimestampFromSeconds(2),
					}.Build(),
				},
			}.Build(),
			storage.Pod_ContainerInstanceList_builder{
				Instances: []*storage.ContainerInstance{
					storage.ContainerInstance_builder{
						InstanceId: storage.ContainerInstanceID_builder{
							Id: "containeridinitfirst",
						}.Build(),
						ContainerName:     "containerinitname",
						ImageDigest:       "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started:           protocompat.GetProtoTimestampFromSeconds(0),
						Finished:          protocompat.GetProtoTimestampFromSeconds(1),
						ExitCode:          137,
						TerminationReason: "Error",
					}.Build(),
					storage.ContainerInstance_builder{
						InstanceId: storage.ContainerInstanceID_builder{
							Id: "containeridinitsecond",
						}.Build(),
						ContainerName: "containerinitname",
						ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started: protocompat.GetProtoTimestampFromSecondsAndNanos(
							1,
							200),

						Finished: protocompat.GetProtoTimestampFromSecondsAndNanos(
							1,
							800),

						ExitCode:          0,
						TerminationReason: "Completed",
					}.Build(),
				},
			}.Build(),
		},
	}.Build()
}

// GetPod1 returns a mock Pod
func GetPod1() *storage.Pod {
	return storage.Pod_builder{
		Id:           fixtureconsts.PodUID1,
		Name:         fixtureconsts.PodName1,
		DeploymentId: GetDeployment().GetId(),
		ClusterId:    "prod cluster",
		Namespace:    "stackrox",
		Started:      protocompat.GetProtoTimestampFromSeconds(0),
		LiveInstances: []*storage.ContainerInstance{
			storage.ContainerInstance_builder{
				InstanceId: storage.ContainerInstanceID_builder{
					Id: "containerid",
				}.Build(),
				ContainerName: "containername",
				ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started:       protocompat.GetProtoTimestampFromSeconds(2),
			}.Build(),
			storage.ContainerInstance_builder{
				InstanceId: storage.ContainerInstanceID_builder{
					Id: "othercontainerid",
				}.Build(),
				ContainerName: "othercontainername",
				ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started:       protocompat.GetProtoTimestampFromSeconds(3),
			}.Build(),
		},
		TerminatedInstances: []*storage.Pod_ContainerInstanceList{
			storage.Pod_ContainerInstanceList_builder{
				Instances: []*storage.ContainerInstance{
					storage.ContainerInstance_builder{
						InstanceId: storage.ContainerInstanceID_builder{
							Id: "containeridfirst",
						}.Build(),
						ContainerName: "containername",
						ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started:       protocompat.GetProtoTimestampFromSeconds(0),
						Finished:      protocompat.GetProtoTimestampFromSeconds(1),
					}.Build(),
				},
			}.Build(),
			storage.Pod_ContainerInstanceList_builder{
				Instances: []*storage.ContainerInstance{
					storage.ContainerInstance_builder{
						InstanceId: storage.ContainerInstanceID_builder{
							Id: "othercontainerid",
						}.Build(),
						ContainerName: "othercontainername",
						ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started:       protocompat.GetProtoTimestampFromSeconds(1),
						Finished:      protocompat.GetProtoTimestampFromSeconds(2),
					}.Build(),
				},
			}.Build(),
			storage.Pod_ContainerInstanceList_builder{
				Instances: []*storage.ContainerInstance{
					storage.ContainerInstance_builder{
						InstanceId: storage.ContainerInstanceID_builder{
							Id: "containeridinitfirst",
						}.Build(),
						ContainerName:     "containerinitname",
						ImageDigest:       "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started:           protocompat.GetProtoTimestampFromSeconds(0),
						Finished:          protocompat.GetProtoTimestampFromSeconds(1),
						ExitCode:          137,
						TerminationReason: "Error",
					}.Build(),
					storage.ContainerInstance_builder{
						InstanceId: storage.ContainerInstanceID_builder{
							Id: "containeridinitsecond",
						}.Build(),
						ContainerName: "containerinitname",
						ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started: protocompat.GetProtoTimestampFromSecondsAndNanos(
							1,
							200),

						Finished: protocompat.GetProtoTimestampFromSecondsAndNanos(
							1,
							800),

						ExitCode:          0,
						TerminationReason: "Completed",
					}.Build(),
				},
			}.Build(),
		},
	}.Build()
}

// GetScopedPod returns a mock Pod belonging to the input scope.
func GetScopedPod(ID string, clusterID string, namespace string) *storage.Pod {
	pod := &storage.Pod{}
	pod.SetId(ID)
	pod.SetClusterId(clusterID)
	pod.SetNamespace(namespace)
	return pod
}
