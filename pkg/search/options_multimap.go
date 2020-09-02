package search

import (
	"strings"
)

// OptionsMultiMap aggregates field name -> search field mappings for multiple options maps.
type OptionsMultiMap interface {
	GetAll(field string) []*Field
}

type optionsMultiMapImpl struct {
	searchFields map[string][]*Field
}

func (m optionsMultiMapImpl) GetAll(field string) []*Field {
	return m.searchFields[strings.ToLower(field)]
}

// MultiMapFromMaps returns an OptionsMultiMap obtained by merging the given maps.
func MultiMapFromMaps(maps ...OptionsMap) OptionsMultiMap {
	result := make(map[string][]*Field)
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
	result := make(map[string][]*Field)
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
