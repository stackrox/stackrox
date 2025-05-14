package clustermetrics

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	metricsPkg "github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/kubernetes/complianceoperator"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	log = logging.LoggerForModule()
	// Interval for querying cluster metrics from Kubernetes and sending to Central.
	defaultInterval = 5 * time.Minute
	// Timeout for querying cluster metrics from Kubernetes.
	defaultTimeout = 10 * time.Second
)

const (
	// coAvailableInUnknownVersion is value used in telemetry
	// when we find Compliance Operator but are unable to extract its version
	coAvailableInUnknownVersion = "CO available in unknown version"
	// coUnavailable is value used in telemetry when we do not find Compliance Operator deployment
	coUnavailable = "CO unavailable"
)

// ClusterMetrics collects metrics from secured clusters and sends them to Central.
type ClusterMetrics interface {
	common.SensorComponent
}

// New returns a new cluster metrics Sensor component.
func New(k8sClient kubernetes.Interface) ClusterMetrics {
	return NewWithInterval(k8sClient, defaultInterval)
}

// NewWithInterval returns a new cluster metrics Sensor component.
func NewWithInterval(k8sClient kubernetes.Interface, pollInterval time.Duration) ClusterMetrics {
	ticker := time.NewTicker(pollInterval)
	ticker.Stop()
	return &clusterMetricsImpl{
		output:          make(chan *message.ExpiringMessage),
		stopper:         concurrency.NewStopper(),
		pollingInterval: pollInterval,
		pollingTimeout:  defaultTimeout,
		k8sClient:       k8sClient,
		pollTicker:      ticker,
	}
}

type clusterMetricsImpl struct {
	coNamespaceCache string

	output          chan *message.ExpiringMessage
	stopper         concurrency.Stopper
	pollingInterval time.Duration
	pollingTimeout  time.Duration
	k8sClient       kubernetes.Interface
	pollTicker      *time.Ticker
}

func (cm *clusterMetricsImpl) Start() error {
	go cm.Poll(cm.pollTicker.C)
	return nil
}

func (cm *clusterMetricsImpl) Stop(_ error) {
	cm.pollTicker.Stop()
	cm.stopper.Client().Stop()
	_ = cm.stopper.Client().Stopped().Wait()
}

func (cm *clusterMetricsImpl) Notify(e common.SensorComponentEvent) {
	switch e {
	case common.SensorComponentEventCentralReachable:
		cm.pollTicker.Reset(cm.pollingInterval)
	case common.SensorComponentEventOfflineMode:
		cm.pollTicker.Stop()
	}
}

func (cm *clusterMetricsImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{}
}

func (cm *clusterMetricsImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (cm *clusterMetricsImpl) ResponsesC() <-chan *message.ExpiringMessage {
	return cm.output
}

func (cm *clusterMetricsImpl) ProcessIndicator(_ *storage.ProcessIndicator) {}

func (cm *clusterMetricsImpl) Poll(tickerC <-chan time.Time) {
	defer cm.stopper.Flow().ReportStopped()

	cm.runPipeline()
	go func() {
		for {
			select {
			case <-cm.stopper.Flow().StopRequested():
				return
			case <-tickerC:
				cm.runPipeline()
			}
		}
	}()
}

func (cm *clusterMetricsImpl) runPipeline() {
	if metrics, err := cm.collectMetrics(); err == nil {
		cm.output <- message.New(&central.MsgFromSensor{
			Msg: &central.MsgFromSensor_ClusterMetrics{
				ClusterMetrics: metrics,
			},
		})
		metricsPkg.SetTelemetryMetrics(metrics)
	} else {
		log.Errorf("Collection of cluster metrics failed: %v", err.Error())
	}
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

	// If cm.coNamespaceCache is empty, the CO deployment will be searched for in all candidate namespaces
	coVersion, namespace, err := complianceoperator.GetInstalledVersion(cm.coNamespaceCache, cm.k8sClient, ctx)
	if err != nil {
		if errors.Is(err, complianceoperator.ErrUnableToExtractVersion) {
			coVersion = coAvailableInUnknownVersion
			cm.coNamespaceCache = namespace
		} else {
			coVersion = coUnavailable
		}
	} else {
		// Remember the CO namespace to accelerate future searches
		cm.coNamespaceCache = namespace
	}
	return &central.ClusterMetrics{NodeCount: nodeCount, CpuCapacity: capacity, CoVersion: coVersion}, nil
}
