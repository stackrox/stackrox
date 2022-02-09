package resources

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoconv"
	v1 "k8s.io/api/core/v1"
)

// serviceAccountDispatcher handles service account events
type serviceAccountDispatcher struct{}

// newServiceAccountDispatcher creates and returns a new service account dispatcher.
func newServiceAccountDispatcher() *serviceAccountDispatcher {
	return &serviceAccountDispatcher{}
}

// ProcessEvent processes a service account resource event, and returns the sensor events to emit in response.
func (*serviceAccountDispatcher) ProcessEvent(obj, _ interface{}, action central.ResourceAction) []*central.SensorEvent {
	serviceAccount := obj.(*v1.ServiceAccount)

	var serviceAccountSecrets []string
	for _, secret := range serviceAccount.Secrets {
		serviceAccountSecrets = append(serviceAccountSecrets, secret.Name)
	}

	var serviceAccountImagePullSecrets []string
	for _, ipSecret := range serviceAccount.ImagePullSecrets {
		serviceAccountImagePullSecrets = append(serviceAccountImagePullSecrets, ipSecret.Name)
	}

	sa := &central.SensorEvent_ServiceAccount{
		ServiceAccount: &storage.ServiceAccount{
			Id:               string(serviceAccount.GetUID()),
			Name:             serviceAccount.GetName(),
			Namespace:        serviceAccount.GetNamespace(),
			ClusterName:      serviceAccount.GetClusterName(),
			CreatedAt:        protoconv.ConvertTimeToTimestamp(serviceAccount.GetCreationTimestamp().Time),
			AutomountToken:   true,
			Labels:           serviceAccount.GetLabels(),
			Annotations:      serviceAccount.GetAnnotations(),
			Secrets:          serviceAccountSecrets,
			ImagePullSecrets: serviceAccountImagePullSecrets,
		},
	}

	if serviceAccount.AutomountServiceAccountToken != nil {
		sa.ServiceAccount.AutomountToken = *serviceAccount.AutomountServiceAccountToken
	}

	return []*central.SensorEvent{
		{
			Id:       string(serviceAccount.UID),
			Action:   action,
			Resource: sa,
		},
	}

}
