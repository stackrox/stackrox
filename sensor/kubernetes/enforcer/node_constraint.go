package enforcer

import (
	"fmt"

	"github.com/stackrox/rox/generated/internalapi/central"
	pkgKubernetes "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/daemonset"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/deployment"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/replicaset"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/replicationcontroller"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/statefulset"
)

func (e *enforcerImpl) unsatisfiableNodeConstraint(enforcement *central.SensorEnforcement) (err error) {
	deploymentInfo := enforcement.GetDeployment()
	if deploymentInfo == nil {
		return fmt.Errorf("unable to apply constraint to non-deployment")
	}

	var function func() error
	switch deploymentInfo.GetDeploymentType() {
	case pkgKubernetes.Deployment:
		function = func() error {
			return deployment.EnforceNodeConstraint(e.client, deploymentInfo)
		}
	case pkgKubernetes.DaemonSet:
		function = func() error {
			return daemonset.EnforceNodeConstraint(e.client, deploymentInfo)
		}
	case pkgKubernetes.ReplicaSet:
		function = func() error {
			return replicaset.EnforceNodeConstraint(e.client, deploymentInfo)
		}
	case pkgKubernetes.ReplicationController:
		function = func() error {
			return replicationcontroller.EnforceNodeConstraint(e.client, deploymentInfo)
		}
	case pkgKubernetes.StatefulSet:
		function = func() error {
			return statefulset.EnforceNodeConstraint(e.client, deploymentInfo)
		}
	default:
		return fmt.Errorf("unknown type: %s", deploymentInfo.GetDeploymentType())
	}

	// Retry any retriable errors encountered when trying to run the enforcement function.
	return retry.WithRetry(function,
		retry.Tries(5),
		retry.OnlyRetryableErrors(),
		retry.OnFailedAttempts(func(e error) {
			log.Error(e)
		}))
}
