package fake

import (
	"time"

	"github.com/stackrox/rox/pkg/kubernetes"
)

func init() {
	workloadRegistry["xlarge"] = xlargeWorkload
}

var (
	xlargeWorkload = &workload{
		DeploymentWorkload: []deploymentWorkload{
			{
				DeploymentType: kubernetes.Deployment,
				NumDeployments: 15000,
				PodWorkload: podWorkload{
					NumPods:           2,
					NumContainers:     3,
					LifecycleDuration: 5 * time.Minute,

					ProcessWorkload: processWorkload{
						ProcessInterval: 30 * time.Second, // deployments * pods / rate = process / second
						AlertRate:       0.001,            // 0.1% of all processes will trigger a runtime alert
					},
				},
				UpdateInterval:    5 * time.Minute,
				LifecycleDuration: 30 * time.Minute,
			},
		},
		NodeWorkload: nodeWorkload{
			NumNodes: 1000,
		},
		NetworkWorkload: networkWorkload{
			FlowInterval: 1 * time.Second,
			BatchSize:    500,
		},
	}
)
