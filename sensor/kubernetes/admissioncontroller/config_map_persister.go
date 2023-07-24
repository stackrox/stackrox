package admissioncontroller

import (
	"compress/gzip"
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/admissioncontrol"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/gziputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
	"github.com/stackrox/rox/sensor/common/message"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	annotationInfoKey  = `stackrox.io/info`
	annotationInfoText = `ConfigMap for admission-control service. Automatically generated - do not modify. Your changes will be overwritten.`
)

var (
	log = logging.LoggerForModule()
)

type configMapPersister struct {
	stopSig concurrency.ErrorSignal

	client v1client.ConfigMapInterface

	settingsStreamIt concurrency.ValueStreamIter[*sensor.AdmissionControlSettings]
}

// NewConfigMapSettingsPersister creates a config persister object for the admission controller.
func NewConfigMapSettingsPersister(k8sClient kubernetes.Interface, settingsMgr admissioncontroller.SettingsManager, namespace string) common.SensorComponent {
	return &configMapPersister{
		client:           k8sClient.CoreV1().ConfigMaps(namespace),
		settingsStreamIt: settingsMgr.SettingsStream().Iterator(false),
	}
}

func (p *configMapPersister) Start() error {
	if !p.stopSig.Reset() {
		return errors.New("config persister was already started")
	}

	go p.run()
	return nil
}

func (p *configMapPersister) Stop(err error) {
	p.stopSig.SignalWithError(err)
}

func (p *configMapPersister) Notify(common.SensorComponentEvent) {}

func (p *configMapPersister) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (p *configMapPersister) ProcessMessage(_ *central.MsgToSensor) error {
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
		log.Errorf("Could not apply admission controller config map: %v", err)
	}

	for !p.stopSig.IsDone() {
		select {
		case <-p.stopSig.Done():
			return

		case <-p.settingsStreamIt.Done():
			p.settingsStreamIt = p.settingsStreamIt.TryNext()

			if err := p.applyCurrentConfigMap(p.ctx()); err != nil {
				log.Errorf("Could not apply admission controller config map: %v", err)
			}
		}
	}
}

func (p *configMapPersister) applyCurrentConfigMap(ctx context.Context) error {
	configMap, err := p.createCurrentConfigMap()
	if err != nil {
		return errors.Wrap(err, "instantiating config map")
	}
	if configMap == nil {
		return nil
	}

	_, err = p.client.Create(ctx, configMap, metav1.CreateOptions{})
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

func settingsToConfigMap(settings *sensor.AdmissionControlSettings) (*v1.ConfigMap, error) {
	clusterConfig := settings.GetClusterConfig()
	enforcedDeployTimePolicies := settings.GetEnforcedDeployTimePolicies()
	runtimePolicies := settings.GetRuntimePolicies()
	if settings == nil || clusterConfig == nil || enforcedDeployTimePolicies == nil || runtimePolicies == nil {
		return nil, nil
	}

	configBytes, err := proto.Marshal(clusterConfig)
	if err != nil {
		return nil, err
	}
	configBytesGZ, err := gziputil.Compress(configBytes, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	deployTimePoliciesBytes, err := proto.Marshal(enforcedDeployTimePolicies)
	if err != nil {
		return nil, errors.Wrap(err, "encoding deploy-time policies")
	}
	deployTimePoliciesBytesGZ, err := gziputil.Compress(deployTimePoliciesBytes, gzip.BestCompression)
	if err != nil {
		return nil, errors.Wrap(err, "compressing deploy-time policies")
	}

	runTimePoliciesBytes, err := proto.Marshal(runtimePolicies)
	if err != nil {
		return nil, errors.Wrap(err, "encoding run-time policies")
	}
	runTimePoliciesBytesGZ, err := gziputil.Compress(runTimePoliciesBytes, gzip.BestCompression)
	if err != nil {
		return nil, errors.Wrap(err, "compressing run-time policies")
	}

	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: admissioncontrol.ConfigMapName,
			Annotations: map[string]string{
				annotationInfoKey: annotationInfoText,
			},
		},
		Data: map[string]string{
			admissioncontrol.LastUpdateTimeDataKey:  settings.GetTimestamp().String(),
			admissioncontrol.CacheVersionDataKey:    settings.GetCacheVersion(),
			admissioncontrol.CentralEndpointDataKey: settings.GetCentralEndpoint(),
			admissioncontrol.ClusterIDDataKey:       settings.GetClusterId(),
		},
		BinaryData: map[string][]byte{
			admissioncontrol.ConfigGZDataKey:             configBytesGZ,
			admissioncontrol.DeployTimePoliciesGZDataKey: deployTimePoliciesBytesGZ,
			admissioncontrol.RunTimePoliciesGZDataKey:    runTimePoliciesBytesGZ,
		},
	}, nil
}

func (p *configMapPersister) createCurrentConfigMap() (*v1.ConfigMap, error) {
	settings := p.settingsStreamIt.Value()
	return settingsToConfigMap(settings)
}
