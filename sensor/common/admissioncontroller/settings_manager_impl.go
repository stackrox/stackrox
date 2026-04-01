package admissioncontroller

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/admissioncontrol"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	pkgPolicies "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/store"
)

type settingsManager struct {
	mutex                         sync.Mutex
	currSettings                  *sensor.AdmissionControlSettings
	settingsStream                *concurrency.ValueStream[*sensor.AdmissionControlSettings]
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
		p.settingsStream.Push(newSettings)
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
		p.settingsStream.Push(newSettings)
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
		p.settingsStream.Push(newSettings)
	}
	p.currSettings = newSettings
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

func (p *settingsManager) InvalidateImageCache(keys []*central.InvalidateImageCache_ImageKey) {
	p.imageCacheInvalidationStream.Push(&sensor.AdmCtrlImageCacheInvalidation{
		ImageKeys: keys,
	})
}
func (p *settingsManager) SettingsStream() concurrency.ReadOnlyValueStream[*sensor.AdmissionControlSettings] {
	return p.settingsStream
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
