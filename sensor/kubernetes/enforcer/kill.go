package enforcer

import (
	"fmt"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/common"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/pod"
)

func (e *enforcerImpl) kill(enforcement *central.SensorEnforcement) (err error) {
	// Fetch the container info, fail if none present as we can only kill containers.
	containerInfo := enforcement.GetContainerInstance()
	if containerInfo == nil {
		return fmt.Errorf("unable to apply constraint to non-deployment")
	}

	// Try to kill the pod containing the container instance.
	err = withReasonableRetry(func() error {
		return pod.EnforceKill(e.client, containerInfo)
	})
	if err != nil {
		return
	}

	// Try to mark the deployment as having the pod killed.
	return withReasonableRetry(func() error {
		return common.MarkPodKilled(e.recorder, getRef(enforcement))
	})
}
