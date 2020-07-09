package fake

import (
	"time"

	"github.com/stackrox/rox/pkg/kubernetes"
)

func init() {
	workloadRegistry["okr-single-load"] = okrSingleLoad
}

var (
	okrSingleLoad = &workload{
		DeploymentWorkload: []deploymentWorkload{
			{
				DeploymentType: kubernetes.Deployment,
				NumDeployments: 10000,
				PodWorkload: podWorkload{
					NumPods:           3,
					NumContainers:     3,
					LifecycleDuration: 24 * time.Hour,

					ProcessWorkload: processWorkload{
						ProcessInterval: 24 * time.Hour, // deployments * pods / rate = process / second
						AlertRate:       0.001,          // 0.1% of all processes will trigger a runtime alert
					},
				},
				UpdateInterval:    24 * time.Hour,
				LifecycleDuration: 24 * time.Hour,
			},
		},
		NodeWorkload: nodeWorkload{
			NumNodes: 1000,
		},
		NetworkWorkload: networkWorkload{
			FlowInterval: 24 * time.Hour,
			BatchSize:    500,
		},
		RBACWorkload: rbacWorkload{
			NumRoles:           5000,
			NumBindings:        5000,
			NumServiceAccounts: 5000,
		},
	}
)
