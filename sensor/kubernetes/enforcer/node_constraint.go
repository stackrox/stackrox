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
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/replicaset"
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/replicationcontroller"
	"github.com/stackrox/stackrox/sensor/kubernetes/enforcer/statefulset"
)

func (e *enforcerImpl) unsatisfiableNodeConstraint(ctx context.Context, enforcement *central.SensorEnforcement) (err error) {
	deploymentInfo := enforcement.GetDeployment()
	if deploymentInfo == nil {
		return errors.New("unable to apply constraint to non-deployment")
	}

	var function func(ctx context.Context) error
	switch deploymentInfo.GetDeploymentType() {
	case pkgKubernetes.Deployment:
		function = func(ctx context.Context) error {
			return deployment.EnforceNodeConstraint(ctx, e.client.Kubernetes(), deploymentInfo)
		}
	case pkgKubernetes.DaemonSet:
		function = func(ctx context.Context) error {
			return daemonset.EnforceNodeConstraint(ctx, e.client.Kubernetes(), deploymentInfo)
		}
	case pkgKubernetes.ReplicaSet:
		function = func(ctx context.Context) error {
			return replicaset.EnforceNodeConstraint(ctx, e.client.Kubernetes(), deploymentInfo)
		}
	case pkgKubernetes.ReplicationController:
		function = func(ctx context.Context) error {
			return replicationcontroller.EnforceNodeConstraint(ctx, e.client.Kubernetes(), deploymentInfo)
		}
	case pkgKubernetes.StatefulSet:
		function = func(ctx context.Context) error {
			return statefulset.EnforceNodeConstraint(ctx, e.client.Kubernetes(), deploymentInfo)
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

	// Mark the deployment as having the node constraint applied.
	return withReasonableRetry(func() error {
		return common.MarkNodeConstraintApplied(e.recorder, enforcement.GetDeployment().GetPolicyName(), getRef(enforcement))
	})
}
