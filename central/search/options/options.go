package options

import (
	"github.com/stackrox/stackrox/central/globalindex"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	searchCommon "github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/set"
)

// CategoryToOptionsSet is a map of all option sets by category, with a category for each indexed data type.
var CategoryToOptionsSet map[v1.SearchCategory]set.StringSet

func generateSetFromOptionsMap(labels []searchCommon.FieldLabel) set.StringSet {
	s := set.NewStringSet()
	for _, l := range labels {
		s.Add(l.String())
	}
	return s
}

// GetOptions returns the searchable fields for the specified categories
func GetOptions(categories []v1.SearchCategory) []string {
	optionsSet := set.NewStringSet()
	for _, category := range categories {
		optionsSet = optionsSet.Union(CategoryToOptionsSet[category])
	}
	return optionsSet.AsSortedSlice(func(optI, optJ string) bool {
		return optI < optJ
	})
}

func init() {
	CategoryToOptionsSet = make(map[v1.SearchCategory]set.StringSet)
	for category, optionsSlice := range globalindex.SearchOptionsMap() {
		CategoryToOptionsSet[category] = generateSetFromOptionsMap(optionsSlice)
	}
}
