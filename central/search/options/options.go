package options

import (
	"sort"

	alertMappings "bitbucket.org/stack-rox/apollo/central/alert/index/mappings"
	deploymentMappings "bitbucket.org/stack-rox/apollo/central/deployment/index/mappings"
	imageMappings "bitbucket.org/stack-rox/apollo/central/image/index/mappings"
	policyMappings "bitbucket.org/stack-rox/apollo/central/policy/index/mappings"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	searchCommon "bitbucket.org/stack-rox/apollo/pkg/search"
	"bitbucket.org/stack-rox/apollo/pkg/set"
	"github.com/deckarep/golang-set"
)

// GlobalOptions is exposed for e2e test
var GlobalOptions = []string{
	searchCommon.Cluster,
	searchCommon.Namespace,
	searchCommon.LabelKey,
	searchCommon.LabelValue,
}

// AllOptionsMaps is a combined map of all search options currently in use.
var AllOptionsMaps = generateAllOptionsMap()

func generateAllOptionsMap() map[string]*v1.SearchField {
	m := make(map[string]*v1.SearchField)
	for k, v := range policyMappings.OptionsMap {
		m[k] = v
	}
	for k, v := range imageMappings.OptionsMap {
		m[k] = v
	}
	for k, v := range deploymentMappings.OptionsMap {
		m[k] = v
	}
	for k, v := range alertMappings.OptionsMap {
		m[k] = v
	}
	return m
}

// CategoryOptionsMap is a map of all option sets by category, with a category for each indexed data type.
var CategoryOptionsMap = map[v1.SearchCategory]mapset.Set{
	v1.SearchCategory_ALERTS:      generateSetFromOptionsMap(alertMappings.OptionsMap, policyMappings.OptionsMap, deploymentMappings.OptionsMap, imageMappings.OptionsMap),
	v1.SearchCategory_POLICIES:    generateSetFromOptionsMap(policyMappings.OptionsMap),
	v1.SearchCategory_DEPLOYMENTS: generateSetFromOptionsMap(deploymentMappings.OptionsMap, imageMappings.OptionsMap),
	v1.SearchCategory_IMAGES:      generateSetFromOptionsMap(imageMappings.OptionsMap),
}

func generateSetFromOptionsMap(maps ...map[string]*v1.SearchField) mapset.Set {
	s := mapset.NewSet()
	for _, m := range maps {
		for k, v := range m {
			if !v.GetHidden() {
				s.Add(k)
			}
		}
	}
	return s
}

// GetOptions returns the searchable fields for the specified categories
func GetOptions(categories []v1.SearchCategory) []string {
	optionsSet := set.NewSetFromStringSlice(GlobalOptions)
	for _, category := range categories {
		optionsSet = optionsSet.Union(CategoryOptionsMap[category])
	}
	slice := set.StringSliceFromSet(optionsSet)
	sort.Strings(slice)
	return slice
}
