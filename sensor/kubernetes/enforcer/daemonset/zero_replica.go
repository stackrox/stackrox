package daemonset

import (
	"context"
	"fmt"

	"github.com/stackrox/stackrox/generated/internalapi/central"
	pkgKub "github.com/stackrox/stackrox/pkg/kubernetes"
	"k8s.io/client-go/kubernetes"
)

// EnforceZeroReplica does nothing but err out, since we can't zero out daemon set replica counts.
func EnforceZeroReplica(_ context.Context, client kubernetes.Interface, deploymentInfo *central.DeploymentEnforcement) (err error) {
	return fmt.Errorf("scaling to 0 is not supported for %s", pkgKub.DaemonSet)
}
