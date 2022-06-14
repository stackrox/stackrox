package enforcer

import (
	"time"

	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/pkg/retry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	clientCoreV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// Helper setup func which creates the event recorder.
func eventRecorder(kubeClient kubernetes.Interface) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&clientCoreV1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: "stackrox/sensor"})
	return recorder
}

// Helper which generates an object reference from the SensorEnforcement input.
func getRef(enforcement *central.SensorEnforcement) *corev1.ObjectReference {
	containerInfo := enforcement.GetContainerInstance()
	if containerInfo != nil {
		return getRefContainer(containerInfo)
	}
	deploymentInfo := enforcement.GetDeployment()
	if deploymentInfo != nil {
		return getRefDeployment(deploymentInfo)
	}
	return nil
}

func getRefContainer(enforcement *central.ContainerInstanceEnforcement) *corev1.ObjectReference {
	return getRefDeployment(enforcement.GetDeploymentEnforcement())
}

func getRefDeployment(enforcement *central.DeploymentEnforcement) *corev1.ObjectReference {
	return &corev1.ObjectReference{
		Kind:       enforcement.GetDeploymentType(),
		APIVersion: "apps/v1",
		Name:       enforcement.GetDeploymentName(),
		Namespace:  enforcement.GetNamespace(),
		UID:        types.UID(enforcement.GetDeploymentId()),
	}
}

// Helper retry function. 5 attempts, 500 millis between attempts, log errors on failures.
func withReasonableRetry(f func() error) error {
	return retry.WithRetry(f,
		retry.Tries(5),
		retry.OnlyRetryableErrors(),
		retry.BetweenAttempts(func(_ int) {
			time.Sleep(500 * time.Millisecond)
		}),
		retry.OnFailedAttempts(func(e error) {
			log.Error(e)
		}))
}
