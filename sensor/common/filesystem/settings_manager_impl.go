package filesystem

import (
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	pkgPolicies "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/configmap"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	configMapName     = "fact-config"
	configMapPathsKey = "paths"

	factConfigFile = "fact.yml"
)

var (
	log = logging.LoggerForModule()
)

type factSettingsManager struct {
	settingsUpdate *concurrency.ValueStream[*v1.ConfigMap]
}

func NewFactSettingsManager() SettingsManager {
	f := &factSettingsManager{
		settingsUpdate: concurrency.NewValueStream[*v1.ConfigMap](nil),
	}

	return f
}

func (f *factSettingsManager) ConfigMapStream() concurrency.ReadOnlyValueStream[*v1.ConfigMap] {
	return f.settingsUpdate
}

func (f *factSettingsManager) UpdateFactSettings(policies []*storage.Policy) {
	paths := f.extractFileActivityPaths(policies)
	if len(paths) == 0 {
		return
	}

	newSettings := &sensor.FactSettings{
		Paths: paths,
	}

	if settings := f.settingsToConfigMap(newSettings); settings != nil {
		f.settingsUpdate.Push(settings)
	}
}

func (f *factSettingsManager) extractFileActivityPaths(policies []*storage.Policy) []string {
	paths := set.NewStringSet()
	for _, policy := range policies {
		if !pkgPolicies.AppliesAtRunTime(policy) ||
			!booleanpolicy.ContainsOneOf(policy, booleanpolicy.FileAccess) {
			// doesn't contain file activity fields, so no paths to extract
			continue
		}

		booleanpolicy.ForEachValueWithFieldName(policy, fieldnames.FilePath, func(value string) bool {
			paths.Add(value)
			return true
		})
	}
	return paths.AsSlice()
}

func (f *factSettingsManager) settingsToConfigMap(settings *sensor.FactSettings) *v1.ConfigMap {
	factConfigYaml, err := yaml.Marshal(settings)
	if err != nil {
		log.Errorf("failed to unmarshal fact settings: %v", err)
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
