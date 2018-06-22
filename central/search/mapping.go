package search

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/set"
	"github.com/deckarep/golang-set"
)

func newStringField(name string) *v1.SearchField {
	return newField(name, v1.SearchDataType_SEARCH_STRING, false)
}

func newBoolField(name string) *v1.SearchField {
	return newField(name, v1.SearchDataType_SEARCH_BOOL, false)
}

func newNumericField(name string) *v1.SearchField {
	return newField(name, v1.SearchDataType_SEARCH_NUMERIC, false)
}

func newSeverityField(name string) *v1.SearchField {
	return newField(name, v1.SearchDataType_SEARCH_SEVERITY, false)
}

func newEnforcementField(name string) *v1.SearchField {
	return newField(name, v1.SearchDataType_SEARCH_ENFORCEMENT, false)
}

func newField(path string, t v1.SearchDataType, store bool) *v1.SearchField {
	return &v1.SearchField{
		Type:      t,
		FieldPath: path,
		Store:     store,
	}
}

// GlobalOptions is exposed for e2e test
var GlobalOptions = []string{
	Cluster,
	Namespace,
	LabelKey,
	LabelValue,
}

// PolicyOptionsMap is exposed for e2e test
var PolicyOptionsMap = map[string]*v1.SearchField{
	Cluster:    newStringField("policy.scope.cluster"),
	Namespace:  newStringField("policy.scope.namespace"),
	LabelKey:   newStringField("policy.scope.label.key"),
	LabelValue: newStringField("policy.scope.label.value"),

	PolicyID:    newStringField("policy.id"),
	Enforcement: newEnforcementField("policy.enforcement"),
	PolicyName:  newStringField("policy.name"),
	Description: newStringField("policy.description"),
	Category:    newStringField("policy.categories"),
	Severity:    newSeverityField("policy.severity"),
}

// ImageOptionsMap is exposed for e2e test
var ImageOptionsMap = map[string]*v1.SearchField{
	CVE:                          newStringField("image.scan.components.vulns.cve"),
	CVSS:                         newNumericField("image.scan.components.vulns.cvss"),
	Component:                    newStringField("image.scan.components.name"),
	DockerfileInstructionKeyword: newStringField("image.metadata.layers.instruction"),
	DockerfileInstructionValue:   newStringField("image.metadata.layers.value"),
	ImageName:                    newStringField("image.name.full_name"),
	ImageSHA:                     newField("image.name.sha", v1.SearchDataType_SEARCH_STRING, true),
	ImageRegistry:                newStringField("image.name.registry"),
	ImageRemote:                  newStringField("image.name.remote"),
	ImageTag:                     newStringField("image.name.tag"),
}

// DeploymentOptionsMap is exposed for e2e test
var DeploymentOptionsMap = map[string]*v1.SearchField{
	Cluster:    newStringField("deployment.cluster_name"),
	Namespace:  newStringField("deployment.namespace"),
	LabelKey:   newStringField("deployment.labels.key"),
	LabelValue: newStringField("deployment.labels.value"),

	CPUCoresLimit:     newNumericField("deployment.containers.resources.cpu_cores_limit"),
	CPUCoresRequest:   newNumericField("deployment.containers.resources.cpu_cores_request"),
	DeploymentID:      newStringField("deployment.id"),
	DeploymentName:    newStringField("deployment.name"),
	DeploymentType:    newStringField("deployment.type"),
	AddCapabilities:   newStringField("deployment.containers.security_context.add_capabilities"),
	DropCapabilities:  newStringField("deployment.containers.security_context.drop_capabilities"),
	EnvironmentKey:    newStringField("deployment.containers.config.env.key"),
	EnvironmentValue:  newStringField("deployment.containers.config.env.value"),
	MemoryLimit:       newNumericField("deployment.containers.resources.memory_mb_limit"),
	MemoryRequest:     newNumericField("deployment.containers.resources.memory_mb_request"),
	Privileged:        newBoolField("deployment.containers.security_context.privileged"),
	SecretName:        newStringField("deployment.containers.secrets.name"),
	SecretPath:        newStringField("deployment.containers.secrets.path"),
	VolumeName:        newStringField("deployment.containers.volumes.name"),
	VolumeSource:      newStringField("deployment.containers.volumes.source"),
	VolumeDestination: newStringField("deployment.containers.volumes.destination"),
	VolumeReadonly:    newBoolField("deployment.containers.volumes.read_only"),
	VolumeType:        newStringField("deployment.containers.volumes.type"),
}

// AlertOptionsMap is exposed for e2e test
var AlertOptionsMap = map[string]*v1.SearchField{
	Violation: newStringField("alert.violations.message"),
	Stale:     newBoolField("alert.stale"),
}

func generateAllOptionsMap() map[string]*v1.SearchField {
	m := make(map[string]*v1.SearchField)
	for k, v := range PolicyOptionsMap {
		m[k] = v
	}
	for k, v := range ImageOptionsMap {
		m[k] = v
	}
	for k, v := range DeploymentOptionsMap {
		m[k] = v
	}
	for k, v := range AlertOptionsMap {
		m[k] = v
	}
	return m
}

var allOptionsMaps = generateAllOptionsMap()

func generateSetFromOptionsMap(maps ...map[string]*v1.SearchField) mapset.Set {
	s := mapset.NewSet()
	for _, m := range maps {
		for k := range m {
			s.Add(k)
		}
	}
	return s
}

var categoryOptionsMap = map[v1.SearchCategory]mapset.Set{
	v1.SearchCategory_ALERTS:      generateSetFromOptionsMap(AlertOptionsMap, PolicyOptionsMap, DeploymentOptionsMap, ImageOptionsMap),
	v1.SearchCategory_POLICIES:    generateSetFromOptionsMap(PolicyOptionsMap),
	v1.SearchCategory_DEPLOYMENTS: generateSetFromOptionsMap(DeploymentOptionsMap, ImageOptionsMap),
	v1.SearchCategory_IMAGES:      generateSetFromOptionsMap(ImageOptionsMap),
}

// GetOptions returns the searchable fields for the specified categories
func GetOptions(categories []v1.SearchCategory) []string {
	optionsSet := set.NewSetFromStringSlice(GlobalOptions)
	for _, category := range categories {
		optionsSet = optionsSet.Union(categoryOptionsMap[category])
	}
	slice := set.StringSliceFromSet(optionsSet)
	sort.Strings(slice)
	return slice
}
