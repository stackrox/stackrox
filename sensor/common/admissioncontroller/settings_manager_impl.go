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
	hasClusterConfig, hasPolicies bool
	centralEndpoint               string

	clusterID clusterIDWaiter

	deployments store.DeploymentStore
	pods        store.PodStore
}

type clusterIDWaiter interface {
	Get() string
}

// NewSettingsManager creates a new settings manager for admission control settings.
func NewSettingsManager(clusterID clusterIDWaiter, deployments store.DeploymentStore, pods store.PodStore) SettingsManager {
	return &settingsManager{
		configStream:       concurrency.NewValueStream[*v1.ConfigMap](nil),
		settingsStream:     concurrency.NewValueStream[*sensor.AdmissionControlSettings](nil),
		sensorEventsStream: concurrency.NewValueStream[*sensor.AdmCtrlUpdateResourceRequest](nil),
		centralEndpoint:    env.CentralEndpoint.Setting(),

		clusterID: clusterID,

		deployments: deployments,
		pods:        pods,
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
		if pkgPolicies.AppliesAtRunTime(policy) &&
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
		p.configStream.Push(config)
	}
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
	return ret
}

func (p *settingsManager) UpdateResources(events ...*central.SensorEvent) {
	for _, event := range events {
		switch event.GetResource().(type) {
		case *central.SensorEvent_Synced, *central.SensorEvent_Deployment, *central.SensorEvent_Pod:
			p.convertAndPush(event)
		case *central.SensorEvent_Namespace:
			// Track namespace deletion to removal sub-resources from admission control.
			if event.GetAction() == central.ResourceAction_REMOVE_RESOURCE {
				p.convertAndPush(event)
			}
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
	clusterConfig := settings.GetClusterConfig()
	enforcedDeployTimePolicies := settings.GetEnforcedDeployTimePolicies()
	runtimePolicies := settings.GetRuntimePolicies()
	if settings == nil || clusterConfig == nil || enforcedDeployTimePolicies == nil || runtimePolicies == nil {
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
