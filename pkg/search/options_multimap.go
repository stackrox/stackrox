package search

import (
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// OptionsMultiMap aggregates field name -> search field mappings for multiple options maps.
type OptionsMultiMap interface {
	GetAll(field string) []*v1.SearchField
}

type optionsMultiMapImpl struct {
	searchFields map[string][]*v1.SearchField
}

func (m optionsMultiMapImpl) GetAll(field string) []*v1.SearchField {
	return m.searchFields[strings.ToLower(field)]
}

// MultiMapFromMaps returns an OptionsMultiMap obtained by merging the given maps.
func MultiMapFromMaps(maps ...OptionsMap) OptionsMultiMap {
	result := make(map[string][]*v1.SearchField)
	for _, m := range maps {
		for label, field := range m.Original() {
			key := strings.ToLower(label.String())
			result[key] = append(result[key], field)
		}
	}
	return optionsMultiMapImpl{
		searchFields: result,
	}
}

// MultiMapFromMapsFiltered returns an OptionsMultiMap obtained by aggregating the given field labels from the
// underlying maps.
func MultiMapFromMapsFiltered(labels []FieldLabel, maps ...OptionsMap) OptionsMultiMap {
	result := make(map[string][]*v1.SearchField)
	for _, m := range maps {
		orig := m.Original()
		for _, label := range labels {
			field := orig[label]
			if field != nil {
				key := strings.ToLower(label.String())
				result[key] = append(result[key], field)
			}
		}
	}
	return optionsMultiMapImpl{
		searchFields: result,
	}
}
