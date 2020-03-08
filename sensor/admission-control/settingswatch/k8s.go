package settingswatch

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/admissioncontrol"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/gziputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/namespaces"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	fieldSelector = fmt.Sprintf("metadata.name=%s", admissioncontrol.ConfigMapName)

	log = logging.LoggerForModule()
)

func tweakListOpts(listOpts *metav1.ListOptions) {
	listOpts.FieldSelector = fieldSelector
}

// WatchK8sForSettingsUpdatesAsync watches kubernetes
func WatchK8sForSettingsUpdatesAsync(ctx concurrency.Waitable, settingsC chan<- *sensor.AdmissionControlSettings) error {
	w := &k8sSettingsWatch{
		ctx:  ctx,
		outC: settingsC,
	}
	return w.start()
}

type k8sSettingsWatch struct {
	ctx  concurrency.Waitable
	outC chan<- *sensor.AdmissionControlSettings
}

func getConfigMapFromObj(obj interface{}) *v1.ConfigMap {
	cm, _ := obj.(*v1.ConfigMap)
	if cm == nil || cm.GetNamespace() != namespaces.StackRox || cm.GetName() != admissioncontrol.ConfigMapName {
		return nil
	}
	return cm
}

func (w *k8sSettingsWatch) OnAdd(obj interface{}) {
	cm := getConfigMapFromObj(obj)
	if cm == nil {
		return
	}

	w.parseAndSendSettings(cm)
}

func (w *k8sSettingsWatch) OnUpdate(oldObj, newObj interface{}) {
	cm := getConfigMapFromObj(newObj)
	if cm == nil {
		return
	}

	w.parseAndSendSettings(cm)
}

func (w *k8sSettingsWatch) OnDelete(oldObj interface{}) {
	w.sendSettings(nil)
}

func parseSettings(cm *v1.ConfigMap) (*sensor.AdmissionControlSettings, error) {
	timestampStr := cm.Data[admissioncontrol.LastUpdateTimeDataKey]
	timestamp, err := time.Parse(time.RFC3339Nano, timestampStr)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse timestamp %q from configmap", timestampStr)
	}
	tsProto, err := types.TimestampProto(timestamp)
	if err != nil {
		return nil, errors.Wrap(err, "timestamp in configmap is not valid")
	}

	policiesGZData := cm.BinaryData[admissioncontrol.PoliciesGZDataKey]
	policiesData, err := gziputil.Decompress(policiesGZData)
	if err != nil {
		return nil, errors.Wrap(err, "could not read gzipped policies data from configmap")
	}

	var policies storage.PolicyList
	if err := proto.Unmarshal(policiesData, &policies); err != nil {
		return nil, errors.Wrap(err, "could not parse protobuf-encoded policies from configmap")
	}

	configGZData := cm.BinaryData[admissioncontrol.ConfigGZDataKey]
	configData, err := gziputil.Decompress(configGZData)
	if err != nil {
		return nil, errors.Wrap(err, "could not read gzipped config data from configmap")
	}

	var config storage.AdmissionControllerConfig
	if err := proto.Unmarshal(configData, &config); err != nil {
		return nil, errors.Wrap(err, "could not parse protobuf-encoded config data from configmap")
	}

	settings := &sensor.AdmissionControlSettings{
		Config:                     &config,
		EnforcedDeployTimePolicies: &policies,
		Timestamp:                  tsProto,
	}

	return settings, nil
}

func (w *k8sSettingsWatch) parseAndSendSettings(cm *v1.ConfigMap) {
	settings, err := parseSettings(cm)
	if err != nil {
		log.Errorf("could not parse admission control configmap: %v", err)
		return
	}
	w.sendSettings(settings)
}

func (w *k8sSettingsWatch) sendSettings(settings *sensor.AdmissionControlSettings) {
	select {
	case <-w.ctx.Done():
	case w.outC <- settings:
	}
}

func (w *k8sSettingsWatch) start() error {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "could not retrieve Kubernetes config")
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return errors.Wrap(err, "could not create kubernetes client")
	}

	sif := informers.NewSharedInformerFactoryWithOptions(k8sClient, 0,
		informers.WithNamespace(namespaces.StackRox),
		informers.WithTweakListOptions(tweakListOpts))

	sif.Core().V1().ConfigMaps().Informer().AddEventHandler(w)
	sif.Start(w.ctx.Done())

	return nil
}
