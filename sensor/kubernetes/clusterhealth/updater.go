package clusterhealth

import (
	"context"
	"errors"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/sensor/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	interval               = 30 * time.Second
	collectorDaemonsetName = "collector"
	collectorContainerName = "collector"
)

var (
	log = logging.LoggerForModule()
)

type updaterImpl struct {
	client  kubernetes.Interface
	updates chan *central.MsgFromSensor
	stopSig concurrency.Signal
}

func (u *updaterImpl) Start() error {
	go u.run()
	return nil
}

func (u *updaterImpl) Stop(_ error) {
	u.stopSig.Signal()
}

func (u *updaterImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.HealthMonitoringCap}
}

func (u *updaterImpl) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (u *updaterImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return u.updates
}

func (u *updaterImpl) run() {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			collectorHealthInfo, err := u.getCollectorInfo()
			if err != nil {
				log.Errorf("Unable to get collector health info: %v", err)
				continue
			}
			select {
			case u.updates <- &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_ClusterHealthInfo{
					ClusterHealthInfo: &central.RawClusterHealthInfo{
						CollectorHealthInfo: collectorHealthInfo,
					},
				},
			}:
				continue
			case <-u.stopSig.Done():
				return
			}
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *updaterImpl) getCollectorInfo() (*storage.CollectorHealthInfo, error) {
	nodes, err := u.client.CoreV1().Nodes().List(u.ctx(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	collectorDS, err := u.client.AppsV1().DaemonSets(namespaces.StackRox).Get(u.ctx(), collectorDaemonsetName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	var version string
	for _, container := range collectorDS.Spec.Template.Spec.Containers {
		if container.Name == collectorContainerName {
			version = stringutils.GetAfterLast(container.Image, ":")
			break
		}
	}

	if version == "" {
		return nil, errors.New("unable to determine collector version")
	}

	return &storage.CollectorHealthInfo{
		Version:              version,
		TotalDesiredPods:     collectorDS.Status.DesiredNumberScheduled,
		TotalReadyPods:       collectorDS.Status.NumberReady,
		TotalRegisteredNodes: int32(len(nodes.Items)),
	}, nil
}

func (u *updaterImpl) ctx() context.Context {
	return concurrency.AsContext(&u.stopSig)
}

// NewUpdater returns a new ready-to-use updater.
func NewUpdater(client kubernetes.Interface) common.SensorComponent {
	return &updaterImpl{
		client:  client,
		updates: make(chan *central.MsgFromSensor),
		stopSig: concurrency.NewSignal(),
	}
}
