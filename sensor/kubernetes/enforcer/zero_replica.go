package enforcer

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	pkgKubernetes "github.com/stackrox/stackrox/pkg/kubernetes"
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/common"
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/daemonset"
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/deployment"
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/deploymentconfig"
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/replicaset"
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/replicationcontroller"
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/statefulset"
)

func (e *enforcerImpl) scaleToZero(ctx context.Context, enforcement *central.SensorEnforcement) (err error) {
	deploymentInfo := enforcement.GetDeployment()
	if deploymentInfo == nil {
		return errors.New("unable to apply constraint to non-deployment")
	}

	// Set enforcement function based on deployment type.
	var function func(ctx context.Context) error
	switch deploymentInfo.GetDeploymentType() {
	case pkgKubernetes.Deployment:
		function = func(ctx context.Context) error {
			return deployment.EnforceZeroReplica(ctx, e.client.Kubernetes(), deploymentInfo)
		}
	case pkgKubernetes.DeploymentConfig:
		function = func(ctx context.Context) error {
			return deploymentconfig.EnforceZeroReplica(ctx, e.client.OpenshiftApps(), deploymentInfo)
		}
	case pkgKubernetes.DaemonSet:
		function = func(ctx context.Context) error {
			return daemonset.EnforceZeroReplica(ctx, e.client.Kubernetes(), deploymentInfo)
		}
	case pkgKubernetes.ReplicaSet:
		function = func(ctx context.Context) error {
			return replicaset.EnforceZeroReplica(ctx, e.client.Kubernetes(), deploymentInfo)
		}
	case pkgKubernetes.ReplicationController:
		function = func(ctx context.Context) error {
			return replicationcontroller.EnforceZeroReplica(ctx, e.client.Kubernetes(), deploymentInfo)
		}
	case pkgKubernetes.StatefulSet:
		function = func(ctx context.Context) error {
			return statefulset.EnforceZeroReplica(ctx, e.client.Kubernetes(), deploymentInfo)
		}
	default:
		return fmt.Errorf("unknown type: %s", deploymentInfo.GetDeploymentType())
	}

	// Retry any retriable errors encountered when trying to run the enforcement function.
	err = withReasonableRetry(func() error {
		return function(ctx)
	})
	if err != nil {
		return
	}

	// Mark the deployment as having been scaled to zero.
	return withReasonableRetry(func() error {
		return common.MarkScaledToZero(e.recorder, enforcement.GetDeployment().GetPolicyName(), getRef(enforcement))
	})
}
