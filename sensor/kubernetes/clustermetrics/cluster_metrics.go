package clustermetrics

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var log = logging.LoggerForModule()

// Interval for querying cluster metrics from Kubernetes and sending to Central.
var defaultInterval = 5 * time.Minute

// Timeout for querying cluster metrics from Kubernetes.
var defaultTimeout = 10 * time.Second

// ClusterMetrics collects metrics from secured clusters and sends them to Central.
type ClusterMetrics interface {
	common.SensorComponent
}

// New returns a new cluster metrics Sensor component.
func New(k8sClient kubernetes.Interface) ClusterMetrics {
	return &clusterMetricsImpl{
		output:          make(chan *central.MsgFromSensor),
		stopper:         concurrency.NewStopper(),
		pollingInterval: defaultInterval,
		pollingTimeout:  defaultTimeout,
		k8sClient:       k8sClient,
	}
}

type clusterMetricsImpl struct {
	output          chan *central.MsgFromSensor
	stopper         concurrency.Stopper
	pollingInterval time.Duration
	pollingTimeout  time.Duration
	k8sClient       kubernetes.Interface
}

func (cm *clusterMetricsImpl) Start() error {
	go cm.Poll()
	return nil
}

func (cm *clusterMetricsImpl) Stop(err error) {
	cm.stopper.Stop()
	cm.stopper.WaitForStopped()
}

func (cm *clusterMetricsImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{}
}

func (cm *clusterMetricsImpl) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (cm *clusterMetricsImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return cm.output
}

func (cm *clusterMetricsImpl) ProcessIndicator(pi *storage.ProcessIndicator) {}

func (cm *clusterMetricsImpl) Poll() {
	defer cm.stopper.Stopped()

	ticker := time.NewTicker(cm.pollingInterval)
	go func() {
		for {
			select {
			case <-cm.stopper.StopDone():
				return
			case <-ticker.C:
				if metrics, err := cm.collectMetrics(); err == nil {
					cm.output <- &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_ClusterMetrics{
							ClusterMetrics: metrics,
						},
					}
				} else {
					log.Errorf("Collection of cluster metrics failed: %v", err.Error())
				}
			}
		}
	}()
}

func (cm *clusterMetricsImpl) collectMetrics() (*central.ClusterMetrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cm.pollingTimeout)
	defer cancel()

	nodes, err := cm.k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var nodeCount int64 = int64(len(nodes.Items))

	var capacity int64
	for _, node := range nodes.Items {
		if cpu := node.Status.Capacity.Cpu(); cpu != nil {
			// Rounds up to the nearest integer away from zero.
			capacity += cpu.Value()
		}
	}

	return &central.ClusterMetrics{NodeCount: nodeCount, CpuCapacity: capacity}, nil
}
