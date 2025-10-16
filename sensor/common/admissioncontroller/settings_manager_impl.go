package admissioncontroller

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/admissioncontrol"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	pkgPolicies "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/store"
	"google.golang.org/protobuf/proto"
)

type settingsManager struct {
	mutex                         sync.Mutex
	currSettings                  *sensor.AdmissionControlSettings
	settingsStream                *concurrency.ValueStream[*sensor.AdmissionControlSettings]
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
	settings.SetClusterId(p.clusterID.Get())
	settings.SetCentralEndpoint(p.centralEndpoint)
	settings.SetTimestamp(protocompat.TimestampNow())
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
	pl := &storage.PolicyList{}
	pl.SetPolicies(deploytimePolicies)
	newSettings.SetEnforcedDeployTimePolicies(pl)
	pl2 := &storage.PolicyList{}
	pl2.SetPolicies(runtimePolicies)
	newSettings.SetRuntimePolicies(pl2)

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
	newSettings.SetClusterConfig(clonedConfig)

	if p.hasClusterConfig && p.hasPolicies {
		p.settingsStream.Push(newSettings)
	}
	p.currSettings = newSettings
}

func (p *settingsManager) FlushCache() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	newSettings := p.newSettingsNoLock()
	newSettings.SetCacheVersion(uuid.NewV4().String())

	if p.hasClusterConfig && p.hasPolicies {
		p.settingsStream.Push(newSettings)
	}
	p.currSettings = newSettings
}

func (p *settingsManager) SettingsStream() concurrency.ReadOnlyValueStream[*sensor.AdmissionControlSettings] {
	return p.settingsStream
}

func (p *settingsManager) SensorEventsStream() concurrency.ReadOnlyValueStream[*sensor.AdmCtrlUpdateResourceRequest] {
	return p.sensorEventsStream
}

func (p *settingsManager) GetResourcesForSync() []*sensor.AdmCtrlUpdateResourceRequest {
	var ret []*sensor.AdmCtrlUpdateResourceRequest
	for _, d := range p.deployments.GetAll() {
		acurr := &sensor.AdmCtrlUpdateResourceRequest{}
		acurr.SetAction(central.ResourceAction_CREATE_RESOURCE)
		acurr.SetDeployment(proto.ValueOrDefault(d))
		ret = append(ret, acurr)
	}

	for _, pod := range p.pods.GetAll() {
		acurr := &sensor.AdmCtrlUpdateResourceRequest{}
		acurr.SetAction(central.ResourceAction_CREATE_RESOURCE)
		acurr.SetPod(proto.ValueOrDefault(pod))
		ret = append(ret, acurr)
	}
	return ret
}

func (p *settingsManager) UpdateResources(events ...*central.SensorEvent) {
	for _, event := range events {
		switch event.WhichResource() {
		case central.SensorEvent_Synced_case, central.SensorEvent_Deployment_case, central.SensorEvent_Pod_case:
			p.convertAndPush(event)
		case central.SensorEvent_Namespace_case:
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
