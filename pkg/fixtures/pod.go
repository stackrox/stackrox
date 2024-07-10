package fixtures

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/protocompat"
)

// GetPod returns a mock Pod
func GetPod() *storage.Pod {
	return &storage.Pod{
		Id:           "nginx-7db9fccd9b-92hfs",
		DeploymentId: GetDeployment().GetId(),
		ClusterId:    "prod cluster",
		Namespace:    "stackrox",
		Started:      protocompat.GetProtoTimestampFromSeconds(0),
		LiveInstances: []*storage.ContainerInstance{
			{
				InstanceId: &storage.ContainerInstanceID{
					Id: "containerid",
				},
				ContainerName: "containername",
				ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started:       protocompat.GetProtoTimestampFromSeconds(2),
			},
			{
				InstanceId: &storage.ContainerInstanceID{
					Id: "othercontainerid",
				},
				ContainerName: "othercontainername",
				ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started:       protocompat.GetProtoTimestampFromSeconds(3),
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
						Started:       protocompat.GetProtoTimestampFromSeconds(0),
						Finished:      protocompat.GetProtoTimestampFromSeconds(1),
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
						Started:       protocompat.GetProtoTimestampFromSeconds(1),
						Finished:      protocompat.GetProtoTimestampFromSeconds(2),
					},
				},
			},
			{
				Instances: []*storage.ContainerInstance{
					{
						InstanceId: &storage.ContainerInstanceID{
							Id: "containeridinitfirst",
						},
						ContainerName:     "containerinitname",
						ImageDigest:       "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started:           protocompat.GetProtoTimestampFromSeconds(0),
						Finished:          protocompat.GetProtoTimestampFromSeconds(1),
						ExitCode:          137,
						TerminationReason: "Error",
					},
					{
						InstanceId: &storage.ContainerInstanceID{
							Id: "containeridinitsecond",
						},
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
		Started:      protocompat.GetProtoTimestampFromSeconds(0),
		LiveInstances: []*storage.ContainerInstance{
			{
				InstanceId: &storage.ContainerInstanceID{
					Id: "containerid",
				},
				ContainerName: "containername",
				ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started:       protocompat.GetProtoTimestampFromSeconds(2),
			},
			{
				InstanceId: &storage.ContainerInstanceID{
					Id: "othercontainerid",
				},
				ContainerName: "othercontainername",
				ImageDigest:   "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started:       protocompat.GetProtoTimestampFromSeconds(3),
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
						Started:       protocompat.GetProtoTimestampFromSeconds(0),
						Finished:      protocompat.GetProtoTimestampFromSeconds(1),
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
						Started:       protocompat.GetProtoTimestampFromSeconds(1),
						Finished:      protocompat.GetProtoTimestampFromSeconds(2),
					},
				},
			},
			{
				Instances: []*storage.ContainerInstance{
					{
						InstanceId: &storage.ContainerInstanceID{
							Id: "containeridinitfirst",
						},
						ContainerName:     "containerinitname",
						ImageDigest:       "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
						Started:           protocompat.GetProtoTimestampFromSeconds(0),
						Finished:          protocompat.GetProtoTimestampFromSeconds(1),
						ExitCode:          137,
						TerminationReason: "Error",
					},
					{
						InstanceId: &storage.ContainerInstanceID{
							Id: "containeridinitsecond",
						},
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

// GetExpectedJSONSerializedTestPod returns the protoJSON serialized form
// of the Deployment returned by GetPod
func GetExpectedJSONSerializedTestPod(_ testing.TB) string {
	return `{
	"id": "nginx-7db9fccd9b-92hfs",
	"deploymentId": "deaaaaaa-bbbb-4011-0000-111111111111",
	"clusterId": "prod cluster",
	"namespace": "stackrox",
	"started": "1970-01-01T00:00:00Z",
	"liveInstances": [
		{
			"instanceId": {"id": "containerid" },
			"containerName": "containername",
			"imageDigest": "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
			"started": "1970-01-01T00:00:02Z"
		},
		{
			"instanceId": {"id": "othercontainerid" },
			"containerName": "othercontainername",
			"imageDigest": "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
			"started": "1970-01-01T00:00:03Z"
		}
	],
	"terminatedInstances": [
		{
			"instances": [
				{
					"instanceId": {"id": "containeridfirst" },
					"containerName": "containername",
					"imageDigest": "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
					"started": "1970-01-01T00:00:00Z",
					"finished": "1970-01-01T00:00:01Z"
				}
			]
		},
		{
			"instances": [
				{
					"instanceId": {"id": "othercontainerid" },
					"containerName": "othercontainername",
					"imageDigest": "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
					"started": "1970-01-01T00:00:01Z",
					"finished": "1970-01-01T00:00:02Z"
				}
			]
		},
		{
			"instances": [
				{
					"instanceId": {"id": "containeridinitfirst" },
					"containerName": "containerinitname",
					"imageDigest": "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
					"started": "1970-01-01T00:00:00Z",
					"finished": "1970-01-01T00:00:01Z",
					"exitCode": 137,
					"terminationReason": "Error"
				},
				{
					"instanceId": {"id": "containeridinitsecond" },
					"containerName": "containerinitname",
					"imageDigest": "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
					"started": "1970-01-01T00:00:01.000000200Z",
					"finished": "1970-01-01T00:00:01.000000800Z",
					"terminationReason": "Completed"
				}
			]
		}
	]
}`
}
