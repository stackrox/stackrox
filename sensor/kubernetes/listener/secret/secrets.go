package secret

import (
	"time"

	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watchlister"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

var logger = logging.LoggerForModule()

// WatchLister implements the WatchLister interface
type WatchLister struct {
	watchlister.WatchLister
	eventC chan<- *listeners.EventWrap
}

// NewWatchLister implements the watch for secrets
func NewWatchLister(client rest.Interface, eventC chan<- *listeners.EventWrap, resyncPeriod time.Duration) *WatchLister {
	npwl := &WatchLister{
		WatchLister: watchlister.NewWatchLister(client, resyncPeriod),
		eventC:      eventC,
	}
	npwl.SetupWatch("secrets", &v1.Secret{}, npwl.resourceChanged)
	return npwl
}

func (npwl *WatchLister) resourceChanged(secretObj interface{}, action pkgV1.ResourceAction) {
	secret, ok := secretObj.(*v1.Secret)
	if !ok {
		logger.Errorf("Object %+v is not a valid secret", secretObj)
		return
	}

	// Filter out service account tokens because we have a service account field.
	// Also filter out DockerConfigJson/DockerCfgs because we don't really care about them.
	switch secret.Type {
	case v1.SecretTypeDockerConfigJson, v1.SecretTypeDockercfg, v1.SecretTypeServiceAccountToken:
		return
	}

	npwl.eventC <- &listeners.EventWrap{
		SensorEvent: &pkgV1.SensorEvent{
			Id:     string(secret.GetUID()),
			Action: action,
			Resource: &pkgV1.SensorEvent_Secret{
				Secret: &pkgV1.Secret{
					Id:          string(secret.GetUID()),
					Name:        secret.GetName(),
					Namespace:   secret.GetNamespace(),
					Type:        string(secret.Type),
					Labels:      secret.GetLabels(),
					Annotations: secret.GetAnnotations(),
				},
			},
		},
	}
}
