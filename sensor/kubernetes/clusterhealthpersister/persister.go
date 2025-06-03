package clusterhealthpersister

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	configMapName      = "sensor-health"
	annotationInfoKey  = `stackrox.io/info`
	annotationInfoText = `ConfigMap for sensor health cluster-local persistence. Automatically generated - do not modify. Your changes will be overwritten.`
	lastUpdatedKey     = "last-updated"
)

var log = logging.LoggerForModule()

type persisterImpl struct {
	// ticker  *time.Ticker
	// tickerC <-chan time.Time

	now func() time.Time

	configMapClient v1client.ConfigMapInterface

	stopper concurrency.Signal
	state   atomic.Value
}

var _ common.SensorComponent = (*persisterImpl)(nil)

func NewClusterHealthPersister(k8sClient kubernetes.Interface, namespace string) common.SensorComponent {
	// ticker := time.NewTicker(10 * time.Second)
	return &persisterImpl{
		// ticker:          ticker,
		// tickerC:         ticker.C,
		now:             time.Now,
		configMapClient: k8sClient.CoreV1().ConfigMaps(namespace),
	}
}

func (p *persisterImpl) Start() error {
	p.state.Store(common.SensorComponentStateSTARTED)
	// p.ticker.Reset(10*time.Second)
	go p.run()
	p.state.Store(common.SensorComponentStateSTARTED)
	return nil
}

func (p *persisterImpl) Stop(error) {
	p.state.Store(common.SensorComponentStateSTOPPING)
	// p.ticker.Stop()
	p.stopper.Signal()
	p.state.Store(common.SensorComponentStateSTOPPED)
}

func (p *persisterImpl) Notify(_ common.SensorComponentEvent) {}

func (p *persisterImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (p *persisterImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (p *persisterImpl) ResponsesC() <-chan *message.ExpiringMessage { return nil }

func (p *persisterImpl) State() common.SensorComponentState {
	return p.state.Load().(common.SensorComponentState)
}

func (p *persisterImpl) run() {
	p.saveHealth()
	for !p.stopper.IsDone() {
		time.Sleep(10 * time.Second)
		p.saveHealth()
	}
	/*
		for {
			select {
			case <-p.tickerC:
				log.Info("Tick")
				p.saveHealth()
			case <-p.stopper.Done():
				return
			}
		}
	*/
}

func (p *persisterImpl) saveHealth() {
	stateReporterMap := common.GetStateReporters()
	stateMap := make(map[string]string, len(stateReporterMap))
	for component, reporter := range stateReporterMap {
		componentState := reporter()
		stateMap[component] = componentState.String()
	}
	stateMap[lastUpdatedKey] = p.now().UTC().Format(time.RFC3339Nano)
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: configMapName,
			Annotations: map[string]string{
				annotationInfoKey: annotationInfoText,
			},
		},
		Data: stateMap,
	}
	_, err := p.configMapClient.Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			log.Error("Failed to save health to config map")
			return
		}
		_, err := p.configMapClient.Update(context.Background(), configMap, metav1.UpdateOptions{})
		if err != nil {
			log.Error("Failed to update health config map")
		}
	}
}
