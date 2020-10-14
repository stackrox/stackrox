package admissioncontroller

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

type settingsManager struct {
	mutex          sync.Mutex
	currSettings   *sensor.AdmissionControlSettings
	settingsStream *concurrency.ValueStream

	hasClusterConfig, hasPolicies bool
	centralEndpoint               string
}

// NewSettingsManager creates a new settings manager for admission control settings.
func NewSettingsManager() SettingsManager {
	return &settingsManager{
		settingsStream:  concurrency.NewValueStream(nil),
		centralEndpoint: env.CentralEndpoint.Setting(),
	}
}

func (p *settingsManager) newSettingsNoLock() *sensor.AdmissionControlSettings {
	settings := &sensor.AdmissionControlSettings{}
	if p.currSettings != nil {
		*settings = *p.currSettings
	}
	settings.CentralEndpoint = p.centralEndpoint
	settings.Timestamp = types.TimestampNow()
	return settings
}

func (p *settingsManager) UpdatePolicies(policies []*storage.Policy) {
	var filtered []*storage.Policy
	for _, policy := range policies {
		if !isEnforcedDeployTimePolicy(policy) {
			continue
		}

		filtered = append(filtered, policy.Clone())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.hasPolicies = true

	newSettings := p.newSettingsNoLock()
	newSettings.EnforcedDeployTimePolicies = &storage.PolicyList{Policies: filtered}

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
