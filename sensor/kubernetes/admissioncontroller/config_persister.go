package admissioncontroller

import (
	"compress/gzip"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/admissioncontrol"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/gziputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/sensor/common/admissioncontroller"
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

type policiesUpdate struct {
	policies []*storage.Policy
	time     time.Time
}

type configUpdate struct {
	config *storage.DynamicClusterConfig
	time   time.Time
}

type configPersister struct {
	stopSig concurrency.ErrorSignal

	client v1client.ConfigMapInterface

	policiesUpdateC    chan policiesUpdate
	policies           []*storage.Policy
	lastPoliciesUpdate time.Time

	configUpdateC    chan configUpdate
	config           *storage.DynamicClusterConfig
	lastConfigUpdate time.Time
}

// NewConfigPersister creates a config persister object for the admission controller.
func NewConfigPersister(k8sClient kubernetes.Interface) admissioncontroller.ConfigPersister {
	return &configPersister{
		client: k8sClient.CoreV1().ConfigMaps(namespaces.StackRox),

		policiesUpdateC: make(chan policiesUpdate),
		configUpdateC:   make(chan configUpdate),
	}
}

func (p *configPersister) Start() error {
	if !p.stopSig.Reset() {
		return errors.New("config persister was already started")
	}

	go p.run()
	return nil
}

func (p *configPersister) Stop(err error) {
	p.stopSig.SignalWithError(err)
}

func (p *configPersister) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (p *configPersister) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (p *configPersister) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

func (p *configPersister) UpdatePolicies(policies []*storage.Policy) {
	var filtered []*storage.Policy
	for _, policy := range policies {
		if !isEnforcedDeployTimePolicy(policy) {
			continue
		}

		filtered = append(filtered, protoutils.CloneStoragePolicy(policy))
	}

	now := time.Now()
	go func() {
		select {
		case p.policiesUpdateC <- policiesUpdate{
			policies: filtered,
			time:     now,
		}:
		case <-p.stopSig.Done():
		}
	}()
}

func (p *configPersister) UpdateConfig(config *storage.DynamicClusterConfig) {
	now := time.Now()
	go func() {
		select {
		case p.configUpdateC <- configUpdate{
			config: config,
			time:   now,
		}:
		case <-p.stopSig.Done():
		}
	}()
}

func (p *configPersister) run() {
	for !p.stopSig.IsDone() {
		select {
		case <-p.stopSig.Done():
			return

		case cfgUpdate := <-p.configUpdateC:
			if !cfgUpdate.time.After(p.lastConfigUpdate) {
				continue
			}

			p.config, p.lastConfigUpdate = cfgUpdate.config, cfgUpdate.time

			if err := p.applyCurrentConfigMap(); err != nil {
				log.Errorf("Could not apply admission controller config map: %v", err)
			}

		case policiesUpdate := <-p.policiesUpdateC:
			if !policiesUpdate.time.After(p.lastPoliciesUpdate) {
				continue
			}

			p.policies, p.lastPoliciesUpdate = policiesUpdate.policies, policiesUpdate.time

			if err := p.applyCurrentConfigMap(); err != nil {
				log.Errorf("Could not apply admission controller config map: %v", err)
			}
		}
	}
}

func (p *configPersister) applyCurrentConfigMap() error {
	configMap, err := p.createCurrentConfigMap()
	if err != nil {
		return errors.Wrap(err, "instantiating config map")
	}

	_, err = p.client.Create(configMap)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return errors.Wrap(err, "telling Kubernetes to create config map")
		}

		if _, err := p.client.Update(configMap); err != nil {
			return errors.Wrap(err, "telling Kubernetes to update existing config map")
		}
	}

	return nil
}

func (p *configPersister) createCurrentConfigMap() (*v1.ConfigMap, error) {
	configBytes, err := proto.Marshal(p.config)
	if err != nil {
		return nil, err
	}
	configBytesGZ, err := gziputil.Compress(configBytes, gzip.BestCompression)
	if err != nil {
		return nil, err
	}

	policiesBytes, err := proto.Marshal(&storage.PolicyList{Policies: p.policies})
	if err != nil {
		return nil, errors.Wrap(err, "encoding policies")
	}
	policiesBytesGZ, err := gziputil.Compress(policiesBytes, gzip.BestCompression)
	if err != nil {
		return nil, errors.Wrap(err, "compressing policies")
	}

	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      admissioncontrol.ConfigMapName,
			Namespace: namespaces.StackRox,
			Annotations: map[string]string{
				annotationInfoKey: annotationInfoText,
			},
		},
		Data: map[string]string{
			admissioncontrol.LastUpdateTimeDataKey: time.Now().Format(time.RFC3339Nano),
		},
		BinaryData: map[string][]byte{
			admissioncontrol.ConfigGZDataKey:   configBytesGZ,
			admissioncontrol.PoliciesGZDataKey: policiesBytesGZ,
		},
	}, nil
}
