package fixtures

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
)

// GetPod returns a mock Pod
func GetPod() *storage.Pod {
	return &storage.Pod{
		Id:           "nginx-7db9fccd9b-92hfs",
		DeploymentId: GetDeployment().GetId(),
		ClusterId:    "prod cluster",
		Namespace:    "stackrox",
		Active:       true,
		Instances: []*storage.ContainerInstance{
			{
				InstanceId: &storage.ContainerInstanceID{
					Id: "containerid",
				},
				ImageDigest: "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started: &types.Timestamp{
					Seconds: 2,
				},
			},
			{
				InstanceId: &storage.ContainerInstanceID{
					Id: "containeridinit",
				},
				ImageDigest: "sha256:035e674c761c8a9bffe25a4f7c552e617869d1c1bfb2f84074c3ee63f3018da4",
				Started: &types.Timestamp{
					Seconds: 0,
				},
				Finished: &types.Timestamp{
					Seconds: 1,
				},
				ExitCode:          0,
				TerminationReason: "Completed",
			},
		},
	}
}
