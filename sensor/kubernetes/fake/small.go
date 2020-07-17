package fake

import (
	"time"

	"github.com/stackrox/rox/pkg/kubernetes"
)

func init() {
	workloadRegistry["small"] = smallWorkload
}

var (
	smallWorkload = &workload{
		DeploymentWorkload: []deploymentWorkload{
			{
				DeploymentType: kubernetes.Deployment,
				NumDeployments: 200,
				PodWorkload: podWorkload{
					NumPods:           5,
					NumContainers:     3,
					LifecycleDuration: 2 * time.Minute,

					ProcessWorkload: processWorkload{
						ProcessInterval: 30 * time.Second, // deployments * pods / rate = process / second
						AlertRate:       0.001,            // 0.1% of all processes will trigger a runtime alert
					},
					ContainerWorkload: containerWorkload{
						NumImages: 0, // 0 => use all images in the fixtures list
					},
				},
				UpdateInterval:    10 * time.Minute,
				LifecycleDuration: 1 * time.Hour,
				NumLifecycles:     0, // 0 => Cycle indefinitely
			},
		},
		NodeWorkload: nodeWorkload{
			NumNodes: 100,
		},
		NetworkWorkload: networkWorkload{
			FlowInterval: 30 * time.Second,
			BatchSize:    100,
		},
		RBACWorkload: rbacWorkload{
			NumRoles:           100,
			NumBindings:        100,
			NumServiceAccounts: 100,
		},
	}
)
