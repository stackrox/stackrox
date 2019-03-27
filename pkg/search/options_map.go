package search

import (
	"fmt"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// HasApplicableOptions determines if a parsed user map has any applicable options to this particular options map
func HasApplicableOptions(specifiedFields []string, optionsMap OptionsMap) bool {
	for _, k := range specifiedFields {
		if _, ok := optionsMap.Get(k); ok {
			return true
		}
	}
	return false
}

// An OptionsMap is a mapping from field labels to search field that permits case-insensitive lookups.
type OptionsMap interface {
	// Get looks for the given string in the OptionsMap. The string is usually user-entered.
	// Get allows case-insensitive lookups.
	Get(field string) (*v1.SearchField, bool)
	// MustGet is used when the values must exist
	MustGet(field string) *v1.SearchField
	// Add adds a search field to the map
	Add(label FieldLabel, field *v1.SearchField) OptionsMap
	// Original returns the original options-map, with cases preserved for FieldLabels.
	// Use this if you need the entire map, with values preserved.
	Original() map[FieldLabel]*v1.SearchField
	// Merge merges two OptionsMaps
	Merge(o OptionsMap) OptionsMap
}

type optionsMapImpl struct {
	normalized map[string]*v1.SearchField
	original   map[FieldLabel]*v1.SearchField
}

func (o *optionsMapImpl) Get(field string) (*v1.SearchField, bool) {
	sf, exists := o.normalized[strings.ToLower(field)]
	return sf, exists
}

func (o *optionsMapImpl) MustGet(field string) *v1.SearchField {
	sf, exists := o.normalized[strings.ToLower(field)]
	if !exists {
		panic(fmt.Sprintf("Could not find field %s in OptionsMap", field))
	}
	return sf
}

func (o *optionsMapImpl) Add(label FieldLabel, field *v1.SearchField) OptionsMap {
	if _, ok := o.original[label]; ok {
		return o
	}
	o.original[label] = field
	o.normalized[strings.ToLower(label.String())] = field
	return o
}

func (o *optionsMapImpl) Original() map[FieldLabel]*v1.SearchField {
	return o.original
}

func (o *optionsMapImpl) Merge(o1 OptionsMap) OptionsMap {
	for k, v := range o1.Original() {
		o.Add(k, v)
	}
	return o
}

// OptionsMapFromMap constructs an OptionsMap object from the given map.
func OptionsMapFromMap(m map[FieldLabel]*v1.SearchField) OptionsMap {
	normalized := make(map[string]*v1.SearchField)
	for k, v := range m {
		normalized[strings.ToLower(string(k))] = v
	}
	return &optionsMapImpl{normalized: normalized, original: m}
}
