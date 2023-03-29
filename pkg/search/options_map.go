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
//
//go:generate mockgen-wrapper
type OptionsMap interface {
	// Get looks for the given string in the OptionsMap. The string is usually user-entered.
	// Get allows case-insensitive lookups.
	Get(field string) (*Field, bool)
	// MustGet is used when the values must exist
	MustGet(field string) *Field
	// Add adds a search field to the map
	Add(label FieldLabel, field *Field) OptionsMap
	// Remove removes a search field from the map
	Remove(label FieldLabel) OptionsMap
	// Original returns the original options-map, with cases preserved for FieldLabels.
	// Use this if you need the entire map, with values preserved.
	Original() map[FieldLabel]*Field
	// Merge merges two OptionsMaps
	Merge(o OptionsMap) OptionsMap
	// PrimaryCategory is the category of the object this options map describes. Note that some of the fields might
	// be linked fields and hence refer to a different category.
	PrimaryCategory() v1.SearchCategory
}

type optionsMapImpl struct {
	normalized      map[string]*Field
	original        map[FieldLabel]*Field
	primaryCategory v1.SearchCategory
}

// NewOptionsMap creates an empty options map
func NewOptionsMap(category v1.SearchCategory) OptionsMap {
	return &optionsMapImpl{
		normalized:      make(map[string]*Field),
		original:        make(map[FieldLabel]*Field),
		primaryCategory: category,
	}
}

func (o *optionsMapImpl) PrimaryCategory() v1.SearchCategory {
	return o.primaryCategory
}

func (o *optionsMapImpl) Get(field string) (*Field, bool) {
	sf, exists := o.normalized[strings.ToLower(field)]
	return sf, exists
}

func (o *optionsMapImpl) MustGet(field string) *Field {
	sf, exists := o.normalized[strings.ToLower(field)]
	if !exists {
		panic(fmt.Sprintf("Could not find field %s in OptionsMap", field))
	}
	return sf
}

func (o *optionsMapImpl) Add(label FieldLabel, field *Field) OptionsMap {
	if _, ok := o.original[label]; ok {
		return o
	}
	o.original[label] = field
	o.normalized[strings.ToLower(label.String())] = field
	return o
}

func (o *optionsMapImpl) Remove(label FieldLabel) OptionsMap {
	delete(o.original, label)
	delete(o.normalized, strings.ToLower(label.String()))
	return o
}

func (o *optionsMapImpl) Original() map[FieldLabel]*Field {
	return o.original
}

func (o *optionsMapImpl) Merge(o1 OptionsMap) OptionsMap {
	for k, v := range o1.Original() {
		o.Add(k, v)
	}
	return o
}

// Difference returns a new option map with all elements of first not in second.
func Difference(o1 OptionsMap, o2 OptionsMap) OptionsMap {
	if o1 == nil {
		return nil
	}

	if o2 == nil {
		return o1
	}

	ret := OptionsMapFromMap(o1.PrimaryCategory(), make(map[FieldLabel]*Field))
	o2.Original()
	for k, v := range o1.Original() {
		if _, ok := o2.Original()[k]; !ok {
			ret.Add(k, v)
		}
	}
	return ret
}

// CombineOptionsMaps does the same thing as Merge, but creates a new map without modifying any inputs.
func CombineOptionsMaps(os ...OptionsMap) OptionsMap {
	if len(os) == 0 {
		return nil
	}
	ret := OptionsMapFromMap(os[0].PrimaryCategory(), make(map[FieldLabel]*Field))
	for _, o := range os {
		ret.Merge(o)
	}
	return ret
}

// OptionsMapFromMap constructs an OptionsMap object from the given map.
func OptionsMapFromMap(primaryCategory v1.SearchCategory, m map[FieldLabel]*Field) OptionsMap {
	normalized := make(map[string]*Field)
	for k, v := range m {
		normalized[strings.ToLower(string(k))] = v
	}
	return &optionsMapImpl{
		normalized:      normalized,
		original:        m,
		primaryCategory: primaryCategory,
	}
}
