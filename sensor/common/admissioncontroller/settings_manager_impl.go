package admissioncontroller

import (
	"compress/gzip"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/admissioncontrol"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/gziputil"
	pkgPolicies "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/configmap"
	"github.com/stackrox/rox/sensor/common/store"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type settingsManager struct {
	mutex                         sync.Mutex
	currSettings                  *sensor.AdmissionControlSettings
	settingsStream                *concurrency.ValueStream[*sensor.AdmissionControlSettings]
	configStream                  *concurrency.ValueStream[*v1.ConfigMap]
	sensorEventsStream            *concurrency.ValueStream[*sensor.AdmCtrlUpdateResourceRequest]
	imageCacheInvalidationStream  *concurrency.ValueStream[*sensor.AdmCtrlImageCacheInvalidation]
	hasClusterConfig, hasPolicies bool
	centralEndpoint               string
	lastClusterLabels             map[string]string

	clusterID     clusterIDWaiter
	clusterLabels clusterLabelsGetter

	deployments store.DeploymentStore
	pods        store.PodStore
	namespaces  store.NamespaceStore
}

type clusterLabelsGetter interface {
	Get() map[string]string
}

type clusterIDWaiter interface {
	Get() string
}

// NewSettingsManager creates a new settings manager for admission control settings.
func NewSettingsManager(clusterID clusterIDWaiter, clusterLabels clusterLabelsGetter, deployments store.DeploymentStore, pods store.PodStore, namespaces store.NamespaceStore) SettingsManager {
	return &settingsManager{
		configStream:                 concurrency.NewValueStream[*v1.ConfigMap](nil),
		settingsStream:               concurrency.NewValueStream[*sensor.AdmissionControlSettings](nil),
		sensorEventsStream:           concurrency.NewValueStream[*sensor.AdmCtrlUpdateResourceRequest](nil),
		imageCacheInvalidationStream: concurrency.NewValueStream[*sensor.AdmCtrlImageCacheInvalidation](nil),
		centralEndpoint:              env.CentralEndpoint.Setting(),

		clusterID:     clusterID,
		clusterLabels: clusterLabels,

		deployments: deployments,
		pods:        pods,
		namespaces:  namespaces,
	}
}

func (p *settingsManager) newSettingsNoLock() *sensor.AdmissionControlSettings {
	settings := &sensor.AdmissionControlSettings{}
	if p.currSettings != nil {
		settings = p.currSettings.CloneVT()
	}
	settings.ClusterId = p.clusterID.Get()
	settings.CentralEndpoint = p.centralEndpoint
	settings.Timestamp = protocompat.TimestampNow()
	if centralcaps.Has(centralsensor.FlattenImageData) {
		settings.FlattenImageData = true
	}
	return settings
}

func (p *settingsManager) UpdatePolicies(policies []*storage.Policy) {
	var deploytimePolicies, runtimePolicies []*storage.Policy
	for _, policy := range policies {
		if isEnforcedDeployTimePolicy(policy) {
			deploytimePolicies = append(deploytimePolicies, policy.CloneVT())
		}
		// Audit log event policies share field types with k8s event policies
		// so ContainsOneOf alone is insufficient to distinguish them.
		if pkgPolicies.AppliesAtRunTime(policy) &&
			policy.GetEventSource() == storage.EventSource_DEPLOYMENT_EVENT &&
			booleanpolicy.ContainsOneOf(policy, booleanpolicy.KubeEvent) {
			runtimePolicies = append(runtimePolicies, policy.CloneVT())
		}
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.hasPolicies = true

	newSettings := p.newSettingsNoLock()
	newSettings.EnforcedDeployTimePolicies = &storage.PolicyList{Policies: deploytimePolicies}
	newSettings.RuntimePolicies = &storage.PolicyList{Policies: runtimePolicies}

	if p.hasClusterConfig && p.hasPolicies {
		p.pushSettings(newSettings)
	}

	p.currSettings = newSettings
}

func (p *settingsManager) UpdateConfig(config *storage.DynamicClusterConfig) {
	clonedConfig := config.CloneVT()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.hasClusterConfig = true

	newSettings := p.newSettingsNoLock()
	newSettings.ClusterConfig = clonedConfig

	if p.hasClusterConfig && p.hasPolicies {
		p.pushSettings(newSettings)
	}
	p.currSettings = newSettings
	p.pushClusterLabelsIfChangedNoLock()
}

func (p *settingsManager) FlushCache() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	newSettings := p.newSettingsNoLock()
	newSettings.CacheVersion = uuid.NewV4().String()

	if p.hasClusterConfig && p.hasPolicies {
		p.pushSettings(newSettings)
	}
	p.currSettings = newSettings
}

func (p *settingsManager) pushSettings(newSettings *sensor.AdmissionControlSettings) {
	p.settingsStream.Push(newSettings)
	if config, err := settingsToConfigMap(newSettings); err != nil {
		log.Errorf("failed to create config map: %v", err)
	} else {
		// the priority is the grpc messages (handled by consumers of the settingsStream)
		// we make no guarantee the config map and the grpc messages will remain in sync.
		// however, under normal operation, failures here or in persisting the config map
		// are unlikely.
		p.configStream.Push(config)
	}
}

// pushClusterLabelsIfChangedNoLock pushes cluster labels to admission control if they've changed.
// Must be called with p.mutex held.
func (p *settingsManager) pushClusterLabelsIfChangedNoLock() {
	if p.clusterLabels == nil {
		return
	}

	currentLabels := p.clusterLabels.Get()
	if mapsEqual(p.lastClusterLabels, currentLabels) {
		return
	}

	p.lastClusterLabels = copyMap(currentLabels)
	p.sensorEventsStream.Push(&sensor.AdmCtrlUpdateResourceRequest{
		Action: central.ResourceAction_SYNC_RESOURCE,
		Resource: &sensor.AdmCtrlUpdateResourceRequest_ClusterLabels{
			ClusterLabels: &sensor.ClusterLabels{Labels: currentLabels},
		},
	})
}

func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func (p *settingsManager) InvalidateImageCache(keys []*central.ImageKey) {
	p.imageCacheInvalidationStream.Push(&sensor.AdmCtrlImageCacheInvalidation{
		ImageKeys: keys,
	})
}
func (p *settingsManager) SettingsStream() concurrency.ReadOnlyValueStream[*sensor.AdmissionControlSettings] {
	return p.settingsStream
}

func (p *settingsManager) ConfigMapStream() concurrency.ReadOnlyValueStream[*v1.ConfigMap] {
	return p.configStream
}

func (p *settingsManager) SensorEventsStream() concurrency.ReadOnlyValueStream[*sensor.AdmCtrlUpdateResourceRequest] {
	return p.sensorEventsStream
}

func (p *settingsManager) ImageCacheInvalidationStream() concurrency.ReadOnlyValueStream[*sensor.AdmCtrlImageCacheInvalidation] {
	return p.imageCacheInvalidationStream
}

func (p *settingsManager) GetResourcesForSync() []*sensor.AdmCtrlUpdateResourceRequest {
	var ret []*sensor.AdmCtrlUpdateResourceRequest
	for _, d := range p.deployments.GetAll() {
		ret = append(ret, &sensor.AdmCtrlUpdateResourceRequest{
			Action: central.ResourceAction_CREATE_RESOURCE,
			Resource: &sensor.AdmCtrlUpdateResourceRequest_Deployment{
				Deployment: d,
			},
		})
	}

	for _, pod := range p.pods.GetAll() {
		ret = append(ret, &sensor.AdmCtrlUpdateResourceRequest{
			Action: central.ResourceAction_CREATE_RESOURCE,
			Resource: &sensor.AdmCtrlUpdateResourceRequest_Pod{
				Pod: pod,
			},
		})
	}

	if p.namespaces != nil {
		for _, ns := range p.namespaces.GetAll() {
			ret = append(ret, &sensor.AdmCtrlUpdateResourceRequest{
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &sensor.AdmCtrlUpdateResourceRequest_Namespace{
					Namespace: ns,
				},
			})
		}
	}

	if p.clusterLabels != nil {
		ret = append(ret, &sensor.AdmCtrlUpdateResourceRequest{
			Action: central.ResourceAction_SYNC_RESOURCE,
			Resource: &sensor.AdmCtrlUpdateResourceRequest_ClusterLabels{
				ClusterLabels: &sensor.ClusterLabels{Labels: p.clusterLabels.Get()},
			},
		})
	}

	return ret
}

func (p *settingsManager) UpdateResources(events ...*central.SensorEvent) {
	for _, event := range events {
		switch event.GetResource().(type) {
		case *central.SensorEvent_Synced, *central.SensorEvent_Deployment, *central.SensorEvent_Pod:
			p.convertAndPush(event)
		case *central.SensorEvent_Namespace:
			// Forward all namespace events so admission control can track namespace labels.
			p.convertAndPush(event)
		}
	}
}

func (p *settingsManager) convertAndPush(event *central.SensorEvent) {
	converted, err := admissioncontrol.SensorEventToAdmCtrlReq(event)
	if err != nil {
		log.Warnf("Ignoring sending sensor event to admission control: %v", err)
	}

	p.sensorEventsStream.Push(converted)
}

func settingsToConfigMap(settings *sensor.AdmissionControlSettings) (*v1.ConfigMap, error) {
	if settings == nil {
		return nil, nil
	}
	clusterConfig := settings.GetClusterConfig()
	enforcedDeployTimePolicies := settings.GetEnforcedDeployTimePolicies()
	runtimePolicies := settings.GetRuntimePolicies()
	if clusterConfig == nil || enforcedDeployTimePolicies == nil || runtimePolicies == nil {
		return nil, nil
	}

	configBytes, err := clusterConfig.MarshalVT()
	if err != nil {
		return nil, errors.Wrap(err, "marshaling cluster config")
	}
	configBytesGZ, err := gziputil.Compress(configBytes, gzip.BestCompression)
	if err != nil {
		return nil, errors.Wrap(err, "compressing cluster config")
	}

	deployTimePoliciesBytes, err := enforcedDeployTimePolicies.MarshalVT()
	if err != nil {
		return nil, errors.Wrap(err, "encoding deploy-time policies")
	}
	deployTimePoliciesBytesGZ, err := gziputil.Compress(deployTimePoliciesBytes, gzip.BestCompression)
	if err != nil {
		return nil, errors.Wrap(err, "compressing deploy-time policies")
	}

	runTimePoliciesBytes, err := runtimePolicies.MarshalVT()
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
			Name:        admissioncontrol.ConfigMapName,
			Annotations: configmap.InfoAnnotations("admission-control"),
		},
		Data: map[string]string{
			admissioncontrol.LastUpdateTimeDataKey:  settings.GetTimestamp().AsTime().Format(time.RFC3339Nano),
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
