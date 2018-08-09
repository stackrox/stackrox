package namespace

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

// NewWatchLister implements the watch for network policies
func NewWatchLister(client rest.Interface, eventC chan<- *listeners.EventWrap, resyncPeriod time.Duration) *WatchLister {
	npwl := &WatchLister{
		WatchLister: watchlister.NewWatchLister(client, resyncPeriod),
		eventC:      eventC,
	}
	npwl.SetupWatch("namespaces", &v1.Namespace{}, npwl.resourceChanged)
	return npwl
}

func (npwl *WatchLister) resourceChanged(namespaceObj interface{}, action pkgV1.ResourceAction) {
	namespace, ok := namespaceObj.(*v1.Namespace)
	if !ok {
		logger.Errorf("Object %+v is not a valid namespace", namespaceObj)
		return
	}
	npwl.eventC <- &listeners.EventWrap{
		SensorEvent: &pkgV1.SensorEvent{
			Id:     string(namespace.GetUID()),
			Action: action,
			Resource: &pkgV1.SensorEvent_Namespace{
				Namespace: &pkgV1.Namespace{
					Id:     string(namespace.GetUID()),
					Name:   namespace.GetName(),
					Labels: namespace.GetLabels(),
				},
			},
		},
	}
}
