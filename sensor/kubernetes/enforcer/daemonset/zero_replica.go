package daemonset

import (
	"fmt"

	roxV1 "github.com/stackrox/rox/generated/api/v1"
	pkgKub "github.com/stackrox/rox/pkg/kubernetes"
	"k8s.io/client-go/kubernetes"
)

// EnforceZeroReplica does nothing but err out, since we can't zero out daemon set replica counts.
func EnforceZeroReplica(client *kubernetes.Clientset, deploymentInfo *roxV1.DeploymentEnforcement) (err error) {
	return fmt.Errorf("scaling to 0 is not supported for %s", pkgKub.DaemonSet)
}
