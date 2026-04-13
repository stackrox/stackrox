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
	"github.com/stackrox/rox/sensor/kubernetes/client"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
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

	dynClient dynamic.Interface
	namespace string

	settingsStreamIt concurrency.ValueStreamIter[*v1.ConfigMap]
}

func NewConfigMapPersister(name, namespace string, dynClient dynamic.Interface, settings concurrency.ValueStreamIter[*v1.ConfigMap]) common.SensorComponent {
	return &configMapPersister{
		name:             name,
		dynClient:        dynClient,
		namespace:        namespace,
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

func (p *configMapPersister) toUnstructured(configMap *v1.ConfigMap) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(configMap)
	if err != nil {
		return nil, errors.Wrap(err, "converting configmap to unstructured")
	}
	return &unstructured.Unstructured{Object: obj}, nil
}

func (p *configMapPersister) applyCurrentConfigMap(ctx context.Context) error {
	configMap := p.settingsStreamIt.Value()
	if configMap == nil {
		return nil
	}

	unstructuredCM, err := p.toUnstructured(configMap)
	if err != nil {
		return err
	}

	cmClient := p.dynClient.Resource(client.ConfigMapGVR).Namespace(p.namespace)

	_, err = cmClient.Create(ctx, unstructuredCM, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "telling Kubernetes to create config map")
		}

		existing, err := cmClient.Get(ctx, configMap.Name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "getting existing config map")
		}
		configMap.ResourceVersion = existing.GetResourceVersion()

		unstructuredCM, err = p.toUnstructured(configMap)
		if err != nil {
			return err
		}

		if _, err := cmClient.Update(ctx, unstructuredCM, metav1.UpdateOptions{}); err != nil {
			return errors.Wrap(err, "telling Kubernetes to update existing config map")
		}
	}

	return nil
}
