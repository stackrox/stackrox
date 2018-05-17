package search

import (
	"sort"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/set"
	"github.com/deckarep/golang-set"
)

func newStringField(name string) *v1.SearchField {
	return newField(name, v1.SearchDataType_SEARCH_STRING)
}

func newBoolField(name string) *v1.SearchField {
	return newField(name, v1.SearchDataType_SEARCH_BOOL)
}

func newNumericField(name string) *v1.SearchField {
	return newField(name, v1.SearchDataType_SEARCH_NUMERIC)
}

func newSeverityField(name string) *v1.SearchField {
	return newField(name, v1.SearchDataType_SEARCH_SEVERITY)
}

func newEnforcementField(name string) *v1.SearchField {
	return newField(name, v1.SearchDataType_SEARCH_ENFORCEMENT)
}

func newField(path string, t v1.SearchDataType) *v1.SearchField {
	return &v1.SearchField{
		Type:      t,
		FieldPath: path,
	}
}

// GlobalOptions is exposed for e2e test
var GlobalOptions = []string{
	"Cluster",
	"Label",
	"Namespace",
}

// PolicyOptionsMap is exposed for e2e test
var PolicyOptionsMap = map[string]*v1.SearchField{
	"Enforcement": newEnforcementField("policy.enforcement"),
	"Policy Name": newStringField("policy.name"),
	"Description": newStringField("policy.description"),
	"Category":    newStringField("policy.categories"),
	"Severity":    newSeverityField("policy.severity"),
}

// ImageOptionsMap is exposed for e2e test
var ImageOptionsMap = map[string]*v1.SearchField{
	"CVE":                            newStringField("image.scan.components.vulns.cve"),
	"CVSS":                           newNumericField("image.scan.components.vulns.cvss"),
	"Component":                      newStringField("image.scan.components.name"),
	"Dockerfile Instruction Keyword": newStringField("image.metadata.layers.instruction"),
	"Dockerfile Instruction Value":   newStringField("image.metadata.layers.value"),
	"Image Name":                     newStringField("image.name.full_name"),
	"Image Registry":                 newStringField("image.name.registry"),
	"Image Remote":                   newStringField("image.name.remote"),
	"Image Tag":                      newStringField("image.name.tag"),
}

// DeploymentOptionsMap is exposed for e2e test
var DeploymentOptionsMap = map[string]*v1.SearchField{
	"Add Capabilities":   newStringField("deployment.containers.security_context.add_capabilities"),
	"Deployment Name":    newStringField("deployment.name"),
	"Deployment Type":    newStringField("deployment.type"),
	"Drop Capabilities":  newStringField("deployment.containers.security_context.drop_capabilities"),
	"Environment Key":    newStringField("deployment.containers.config.env.key"),
	"Environment Value":  newStringField("deployment.containers.config.env.value"),
	"Privileged":         newBoolField("deployment.containers.security_context.privileged"),
	"Secret Name":        newStringField("deployment.containers.secrets.name"),
	"Secret Path":        newStringField("deployment.containers.secrets.path"),
	"Volume Name":        newStringField("deployment.containers.volumes.name"),
	"Volume Source":      newStringField("deployment.containers.volumes.source"),
	"Volume Destination": newStringField("deployment.containers.volumes.destination"),
	"Volume ReadOnly":    newBoolField("deployment.containers.volumes.read_only"),
	"Volume Type":        newStringField("deployment.containers.volumes.type"),
}

// AlertOptionsMap is exposed for e2e test
var AlertOptionsMap = map[string]*v1.SearchField{
	"Violation": newStringField("alert.violations.message"),
	"Stale":     newBoolField("alert.stale"),
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
