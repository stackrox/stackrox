package lightspeed

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/unimplemented"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	olsAPIGroup     = "ols.openshift.io"
	olsAPIVersion   = "v1alpha1"
	olsGroupVersion = olsAPIGroup + "/" + olsAPIVersion
	olsPartOfLabel  = "app.kubernetes.io/part-of=openshift-lightspeed"
	updateInterval  = 30 * time.Second
)

var (
	log = logging.LoggerForModule()

	olsConfigGVR = schema.GroupVersionResource{
		Group:    olsAPIGroup,
		Version:  olsAPIVersion,
		Resource: "olsconfigs",
	}
)

// NewUpdater creates a Sensor component that periodically detects OpenShift Lightspeed.
func NewUpdater(k8sClient client.Interface) common.SensorComponent {
	ticker := time.NewTicker(updateInterval)
	ticker.Stop()
	return &updater{
		client:   k8sClient,
		response: make(chan *message.ExpiringMessage),
		stopSig:  concurrency.NewSignal(),
		ticker:   ticker,
	}
}

type updater struct {
	unimplemented.Receiver

	client   client.Interface
	response chan *message.ExpiringMessage
	stopSig  concurrency.Signal
	ticker   *time.Ticker
}

func (u *updater) Name() string {
	return "lightspeed.updater"
}

func (u *updater) Start() error {
	go u.run()
	return nil
}

func (u *updater) Stop() {
	u.ticker.Stop()
	u.stopSig.Signal()
}

func (u *updater) Notify(e common.SensorComponentEvent) {
	log.Info(common.LogSensorComponentEvent(e))
	switch e {
	case common.SensorComponentEventSyncFinished:
		u.ticker.Reset(updateInterval)
	case common.SensorComponentEventOfflineMode:
		u.ticker.Stop()
	}
}

func (u *updater) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{}
}

func (u *updater) ResponsesC() <-chan *message.ExpiringMessage {
	return u.response
}

func (u *updater) run() {
	u.detectAndSend()

	for {
		select {
		case <-u.ticker.C:
			u.detectAndSend()
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *updater) detectAndSend() {
	info := u.detect()

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_LightspeedInfo{
			LightspeedInfo: info,
		},
	}

	select {
	case u.response <- message.New(msg):
	case <-u.stopSig.Done():
	}
}

func (u *updater) detect() *central.LightspeedInfo {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Step 1: Check if the OLSConfig CRD exists via Discovery API.
	_, err := u.client.Kubernetes().Discovery().ServerResourcesForGroupVersion(olsGroupVersion)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return &central.LightspeedInfo{IsAvailable: false}
		}
		return &central.LightspeedInfo{
			IsAvailable: false,
			StatusError: fmt.Sprintf("checking OLS CRD: %v", err),
		}
	}

	// Step 2: Check if any OLSConfig CR instances exist using the dynamic client.
	crList, err := u.client.Dynamic().Resource(olsConfigGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return &central.LightspeedInfo{
			IsAvailable: false,
			StatusError: fmt.Sprintf("listing OLSConfig resources: %v", err),
		}
	}
	if len(crList.Items) == 0 {
		return &central.LightspeedInfo{IsAvailable: false}
	}

	// Step 3: Find the Lightspeed service by label selector.
	svcList, err := u.client.Kubernetes().CoreV1().Services("").List(ctx, metav1.ListOptions{
		LabelSelector: olsPartOfLabel,
	})
	if err != nil {
		return &central.LightspeedInfo{
			IsAvailable: false,
			StatusError: fmt.Sprintf("finding Lightspeed service: %v", err),
		}
	}
	if len(svcList.Items) == 0 {
		return &central.LightspeedInfo{
			IsAvailable: false,
			StatusError: "OLSConfig exists but no service with label app.kubernetes.io/part-of=openshift-lightspeed found",
		}
	}

	svc := svcList.Items[0]
	var port int32
	for _, p := range svc.Spec.Ports {
		if p.Name == "https" {
			port = p.Port
			break
		}
	}
	if port == 0 && len(svc.Spec.Ports) > 0 {
		port = svc.Spec.Ports[0].Port
	}
	if port == 0 {
		return &central.LightspeedInfo{
			IsAvailable: false,
			StatusError: "Lightspeed service has no ports defined",
		}
	}

	endpoint := fmt.Sprintf("https://%s.%s.svc:%d", svc.Name, svc.Namespace, port)
	log.Debugf("OpenShift Lightspeed detected at %s", endpoint)

	return &central.LightspeedInfo{
		IsAvailable: true,
		Endpoint:    endpoint,
	}
}
