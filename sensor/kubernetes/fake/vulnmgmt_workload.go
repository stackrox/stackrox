package fake

import (
	"time"

	"github.com/stackrox/rox/pkg/kubernetes"
)

func init() {
	workloadRegistry["vulnmgmt"] = vulnMgmtWorkload
}

var (
	vulnMgmtWorkload = &workload{
		DeploymentWorkload: []deploymentWorkload{
			{
				DeploymentType: kubernetes.Deployment,
				NumDeployments: 2500,
				PodWorkload: podWorkload{
					NumPods:           3,
					NumContainers:     3,
					LifecycleDuration: 10 * time.Minute,

					ProcessWorkload: processWorkload{
						ProcessInterval: 0,     // deployments * pods / rate = process / second
						AlertRate:       0.001, // 0.1% of all processes will trigger a runtime alert
					},
					ContainerWorkload: containerWorkload{
						NumImages: 0, // 0 => use all images in the fixtures list
					},
				},
				UpdateInterval:    10 * time.Minute,
				LifecycleDuration: 60 * time.Minute,
				NumLifecycles:     0, // 0 => Cycle indefinitely
			},
		},
		NodeWorkload: nodeWorkload{
			NumNodes: 1000,
		},
		NetworkWorkload: networkWorkload{
			FlowInterval: 0,
			BatchSize:    100,
		},
		RBACWorkload: rbacWorkload{
			NumRoles:           1000,
			NumBindings:        1000,
			NumServiceAccounts: 1000,
		},
	}
)
