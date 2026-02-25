package configmap

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/unimplemented"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	annotationInfoKey     = `stackrox.io/info`
	annotationInfoTextFmt = `ConfigMap for %s service. Automatically generated - do not modify. Your changes will be overwritten.`
)

var (
	log = logging.LoggerForModule()
)

func InfoAnnotations(serviceName string) map[string]string {
	return map[string]string{
		annotationInfoKey: fmt.Sprintf(annotationInfoTextFmt, serviceName),
	}
}

type configMapPersister struct {
	unimplemented.Receiver

	name string

	stopSig concurrency.ErrorSignal

	client v1client.ConfigMapInterface

	settingsStreamIt concurrency.ValueStreamIter[*v1.ConfigMap]
}

func NewConfigMapPersister(name, namespace string, k8s kubernetes.Interface, settings concurrency.ValueStreamIter[*v1.ConfigMap]) common.SensorComponent {
	return &configMapPersister{
		name:             name,
		client:           k8s.CoreV1().ConfigMaps(namespace),
		settingsStreamIt: settings,
	}
}

func (p *configMapPersister) Start() error {
	if !p.stopSig.Reset() {
		return errors.New("config persister was already started")
	}

	go p.run()
	return nil
}

func (p *configMapPersister) Name() string {
	return fmt.Sprintf("%s.configMapPersister", p.name)
}

func (p *configMapPersister) Stop() {
	p.stopSig.Signal()
}

func (p *configMapPersister) Notify(common.SensorComponentEvent) {}

func (p *configMapPersister) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (p *configMapPersister) ResponsesC() <-chan *message.ExpiringMessage {
	return nil
}

func (p *configMapPersister) ctx() context.Context {
	return concurrency.AsContext(&p.stopSig)
}

func (p *configMapPersister) run() {
	// Attempt to apply the initial config, if any.
	if err := p.applyCurrentConfigMap(p.ctx()); err != nil {
		log.Errorf("Could not apply %s config map: %v", p.name, err)
	}

	for !p.stopSig.IsDone() {
		select {
		case <-p.stopSig.Done():
			return

		case <-p.settingsStreamIt.Done():
			p.settingsStreamIt = p.settingsStreamIt.TryNext()

			if err := p.applyCurrentConfigMap(p.ctx()); err != nil {
				log.Errorf("Could not apply %s config map: %v", p.name, err)
			}
		}
	}
}

func (p *configMapPersister) applyCurrentConfigMap(ctx context.Context) error {
	configMap := p.settingsStreamIt.Value()
	if configMap == nil {
		return nil
	}

	_, err := p.client.Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "telling Kubernetes to create config map")
		}

		if _, err := p.client.Update(ctx, configMap, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "telling Kubernetes to update existing config map")
		}
	}

	return nil
}
