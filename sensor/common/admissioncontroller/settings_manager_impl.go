package admissioncontroller

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

type settingsManager struct {
	mutex          sync.Mutex
	currSettings   *sensor.AdmissionControlSettings
	settingsStream *concurrency.ValueStream

	hasClusterConfig, hasPolicies bool
}

// NewSettingsManager creates a new settings manager for admission control settings.
func NewSettingsManager() SettingsManager {
	return &settingsManager{
		settingsStream: concurrency.NewValueStream(nil),
	}
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

	newSettings := &sensor.AdmissionControlSettings{
		ClusterConfig:              p.currSettings.GetClusterConfig(),
		EnforcedDeployTimePolicies: &storage.PolicyList{Policies: filtered},
		Timestamp:                  types.TimestampNow(),
	}

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

	newSettings := &sensor.AdmissionControlSettings{
		ClusterConfig:              clonedConfig,
		EnforcedDeployTimePolicies: p.currSettings.GetEnforcedDeployTimePolicies(),
		Timestamp:                  types.TimestampNow(),
	}

	if p.hasClusterConfig && p.hasPolicies {
		p.settingsStream.Push(newSettings)
	}
	p.currSettings = newSettings

}

func (p *settingsManager) SettingsStream() concurrency.ReadOnlyValueStream {
	return p.settingsStream
}
