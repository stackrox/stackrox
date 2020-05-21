package fake

import (
	"time"

	"github.com/stackrox/rox/pkg/kubernetes"
)

func init() {
	workloadRegistry["long-running"] = longRunningWorkload
}

var (
	longRunningWorkload = &workload{
		DeploymentWorkload: []deploymentWorkload{
			{
				DeploymentType: kubernetes.Deployment,
				NumDeployments: 1000,
				PodWorkload: podWorkload{
					NumPods:           5,
					NumContainers:     3,
					LifecycleDuration: 2 * time.Minute,

					ProcessWorkload: processWorkload{
						ProcessInterval: 30 * time.Second, // deployments * pods / rate = process / second
						AlertRate:       0.005,            // 0.5% of all processes will trigger a runtime alert
					},
				},
				UpdateInterval:    100 * time.Second,
				LifecycleDuration: 10 * time.Minute,
			},
		},
		NodeWorkload: nodeWorkload{
			NumNodes: 1000,
		},
		NetworkWorkload: networkWorkload{
			FlowInterval: 1 * time.Second,
			BatchSize:    100,
		},
	}
)
