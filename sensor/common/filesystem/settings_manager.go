package filesystem

import (
	"slices"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	pkgPolicies "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/configmap"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	configMapName  = "fact-config"
	factConfigFile = "fact.yml"
)

var (
	log = logging.LoggerForModule()
)

// FactSettingsManager extracts file activity paths from policies and
// publishes them as a ConfigMap for Fact to consume.
type FactSettingsManager struct {
	mutex          sync.Mutex
	settingsUpdate *concurrency.ValueStream[*v1.ConfigMap]
	lastPaths      set.StringSet
}

func NewFactSettingsManager() *FactSettingsManager {
	return &FactSettingsManager{
		settingsUpdate: concurrency.NewValueStream[*v1.ConfigMap](nil),
	}
}

func (f *FactSettingsManager) ConfigMapStream() concurrency.ReadOnlyValueStream[*v1.ConfigMap] {
	return f.settingsUpdate
}

func (f *FactSettingsManager) UpdateFactSettings(policies []*storage.Policy) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	paths := f.extractFileActivityPaths(policies)
	if paths.Equal(f.lastPaths) {
		return
	}
	f.lastPaths = paths

	pathSlice := paths.AsSlice()
	slices.Sort(pathSlice)
	newSettings := &sensor.FactSettings{Paths: pathSlice}

	if settings := f.settingsToConfigMap(newSettings); settings != nil {
		f.settingsUpdate.Push(settings)
	}
}

func (f *FactSettingsManager) extractFileActivityPaths(policies []*storage.Policy) set.StringSet {
	paths := set.NewStringSet()
	for _, policy := range policies {
		if !isActiveFileAccessPolicy(policy) {
			continue
		}

		booleanpolicy.ForEachValueWithFieldName(policy, fieldnames.FilePath, func(value string) bool {
			paths.Add(value)
			return true
		})
	}
	return paths
}

func isActiveFileAccessPolicy(policy *storage.Policy) bool {
	return !policy.GetDisabled() &&
		pkgPolicies.AppliesAtRunTime(policy) &&
		booleanpolicy.ContainsOneOf(policy, booleanpolicy.FileAccess) &&
		(policy.GetEventSource() == storage.EventSource_DEPLOYMENT_EVENT ||
			policy.GetEventSource() == storage.EventSource_NODE_EVENT)
}

func (f *FactSettingsManager) settingsToConfigMap(settings *sensor.FactSettings) *v1.ConfigMap {
	factConfigYaml, err := yaml.Marshal(settings)
	if err != nil {
		log.Errorf("failed to marshal fact settings: %v", err)
		return nil
	}

	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        configMapName,
			Annotations: configmap.InfoAnnotations("fact"),
		},
		Data: map[string]string{
			factConfigFile: string(factConfigYaml),
		},
	}
}
