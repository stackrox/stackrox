package search

import (
	"strings"

	"github.com/stackrox/rox/generated/api/v1"
)

// An OptionsMap is a mapping from field labels to search field that permits case-insensitive lookups.
type OptionsMap interface {
	// Get looks for the given string in the OptionsMap. The string is usually user-entered.
	// Get allows case-insensitive lookups.
	Get(field string) (*v1.SearchField, bool)
	// Original returns the original options-map, with cases preserved for FieldLabels.
	// Use this if you need the entire map, with values preserved.
	Original() map[FieldLabel]*v1.SearchField
}

type optionsMapImpl struct {
	normalized map[string]*v1.SearchField
	original   map[FieldLabel]*v1.SearchField
}

func (o *optionsMapImpl) Get(field string) (*v1.SearchField, bool) {
	sf, exists := o.normalized[strings.ToLower(field)]
	return sf, exists
}

func (o *optionsMapImpl) Original() map[FieldLabel]*v1.SearchField {
	return o.original
}

// OptionsMapFromMap constructs an OptionsMap object from the given map.
func OptionsMapFromMap(m map[FieldLabel]*v1.SearchField) OptionsMap {
	normalized := make(map[string]*v1.SearchField)
	for k, v := range m {
		normalized[strings.ToLower(string(k))] = v
	}
	return &optionsMapImpl{normalized: normalized, original: m}
}
