package admissioncontroller

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/internalapi/sensor"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/admissioncontrol"
	"github.com/stackrox/stackrox/pkg/booleanpolicy"
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/env"
	pkgPolicies "github.com/stackrox/stackrox/pkg/policies"
	"github.com/stackrox/stackrox/pkg/sync"
	"github.com/stackrox/stackrox/pkg/uuid"
	"github.com/stackrox/stackrox/sensor/common/clusterid"
	"github.com/stackrox/stackrox/sensor/common/store"
)

type settingsManager struct {
	mutex                         sync.Mutex
	currSettings                  *sensor.AdmissionControlSettings
	settingsStream                *concurrency.ValueStream
	sensorEventsStream            *concurrency.ValueStream
	hasClusterConfig, hasPolicies bool
	centralEndpoint               string

	deployments store.DeploymentStore
	pods        store.PodStore
}

// NewSettingsManager creates a new settings manager for admission control settings.
func NewSettingsManager(deployments store.DeploymentStore, pods store.PodStore) SettingsManager {
	return &settingsManager{
		settingsStream:     concurrency.NewValueStream(nil),
		sensorEventsStream: concurrency.NewValueStream(nil),
		centralEndpoint:    env.CentralEndpoint.Setting(),

		deployments: deployments,
		pods:        pods,
	}
}

func (p *settingsManager) newSettingsNoLock() *sensor.AdmissionControlSettings {
	settings := &sensor.AdmissionControlSettings{}
	if p.currSettings != nil {
		*settings = *p.currSettings
	}
	settings.ClusterId = clusterid.Get()
	settings.CentralEndpoint = p.centralEndpoint
	settings.Timestamp = types.TimestampNow()
	return settings
}

func (p *settingsManager) UpdatePolicies(policies []*storage.Policy) {
	var deploytimePolicies, runtimePolicies []*storage.Policy
	for _, policy := range policies {
		if isEnforcedDeployTimePolicy(policy) {
			deploytimePolicies = append(deploytimePolicies, policy.Clone())
		}
		if pkgPolicies.AppliesAtRunTime(policy) &&
			booleanpolicy.ContainsOneOf(policy, booleanpolicy.KubeEvent) {
			runtimePolicies = append(runtimePolicies, policy.Clone())
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
	clonedConfig := config.Clone()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.hasClusterConfig = true

	newSettings := p.newSettingsNoLock()
	newSettings.ClusterConfig = clonedConfig

	if p.hasClusterConfig && p.hasPolicies {
		p.settingsStream.Push(newSettings)
	}
	p.currSettings = newSettings
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

func (p *settingsManager) SettingsStream() concurrency.ReadOnlyValueStream {
	return p.settingsStream
}

func (p *settingsManager) SensorEventsStream() concurrency.ReadOnlyValueStream {
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
