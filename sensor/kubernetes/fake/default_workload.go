package fake

import (
	"time"

	"github.com/stackrox/rox/pkg/kubernetes"
)

func init() {
	workloadRegistry["default"] = defaultWorkload
}

var (
	defaultWorkload = &workload{
		DeploymentWorkload: []deploymentWorkload{
			{
				DeploymentType: kubernetes.Deployment,
				NumDeployments: 2500,
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
				UpdateInterval:    100 * time.Second,
				LifecycleDuration: 10 * time.Minute,
				NumLifecycles:     0, // 0 => Cycle indefinitely
			},
		},
		NodeWorkload: nodeWorkload{
			NumNodes: 1000,
		},
		NetworkWorkload: networkWorkload{
			FlowInterval: 1 * time.Second,
			BatchSize:    100,
		},
		RBACWorkload: rbacWorkload{
			NumRoles:           1000,
			NumBindings:        1000,
			NumServiceAccounts: 1000,
		},
	}
)
