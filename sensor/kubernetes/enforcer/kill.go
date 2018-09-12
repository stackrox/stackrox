package enforcer

import (
	"fmt"
	"time"

	roxV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/sensor/kubernetes/enforcer/pod"
)

func (e *enforcerImpl) kill(enforcement *roxV1.SensorEnforcement) (err error) {
	// Fetch the container info, fail if none present as we can only kill containers.
	containerInfo := enforcement.GetContainerInstance()
	if containerInfo == nil {
		return fmt.Errorf("unable to apply constraint to non-deployment")
	}

	// Try to kill the pod containing the container instance.
	function := func() error {
		return pod.EnforceKill(e.client, containerInfo)
	}

	// Retry any retryable errors encountered when trying to run the enforcement function.
	return retry.WithRetry(function,
		retry.Tries(5),
		retry.OnlyRetryableErrors(),
		retry.BetweenAttempts(func() {
			time.Sleep(time.Second)
		}),
		retry.OnFailedAttempts(func(e error) {
			logger.Error(e)
		}))
}
