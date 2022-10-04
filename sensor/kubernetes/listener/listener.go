package listener

import (
	"context"
	"io"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/awscredentials"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	log = logging.LoggerForModule()
)

// New returns a new kubernetes listener.
func New(client client.Interface, configHandler config.Handler, detector detector.Detector, nodeName string, resyncPeriod time.Duration, traceWriter io.Writer) common.SensorComponent {
	k := &listenerImpl{
		client:             client,
		eventsC:            make(chan *central.MsgFromSensor, 10),
		stopSig:            concurrency.NewSignal(),
		configHandler:      configHandler,
		detector:           detector,
		credentialsManager: createCredentialsManager(client, nodeName),
		resyncPeriod:       resyncPeriod,
		traceWriter:        traceWriter,
	}
	return k
}

// createCredentialsManager retrieves Sensor's node provider ID and creates an AWS credentials manager.
func createCredentialsManager(client client.Interface, nodeName string) (credentialsManager awscredentials.RegistryCredentialsManager) {
	node, err := client.Kubernetes().CoreV1().Nodes().Get(
		context.Background(), nodeName, metav1.GetOptions{})
	if err != nil {
		log.Warnf("ECR credential manager is not available: failed to read node provider: %v", err)
		return
	}
	credentialsManager, err = awscredentials.NewECRCredentialsManager(node.Spec.ProviderID)
	if err != nil {
		log.Warnf("ECR credential manager is not available: %v", err)
	}
	return
}
