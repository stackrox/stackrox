package enforcer

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/common"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/pod"
)

func (e *enforcerImpl) kill(ctx context.Context, enforcement *central.SensorEnforcement) error {
	// Fetch the container info, fail if none present as we can only kill containers.
	containerInfo := enforcement.GetContainerInstance()
	if containerInfo == nil {
		return errors.New("unable to apply constraint to non-deployment")
	}

	// Try to kill the pod containing the container instance.
	var enforcementExecuted bool
	err := withReasonableRetry(func() error {
		var err error
		enforcementExecuted, err = pod.EnforceKill(ctx, e.client.Kubernetes(), containerInfo)
		return err
	})
	if err != nil {
		return err
	}
	if !enforcementExecuted {
		return nil
	}

	// Try to mark the deployment as having the pod killed.
	return withReasonableRetry(func() error {
		return common.MarkPodKilled(e.recorder, enforcement.GetContainerInstance().GetPodId(), enforcement.GetContainerInstance().GetDeploymentEnforcement().GetPolicyName(), getRef(enforcement))
	})
}
